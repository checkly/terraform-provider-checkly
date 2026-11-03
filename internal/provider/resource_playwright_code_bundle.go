package provider

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tfpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	checkly "github.com/checkly/checkly-go-sdk"
)

const (
	playwrightPrebuiltArchiveAttributeName = "prebuilt_archive"
	playwrightMetadataAttributeName        = "metadata"
)

var (
	_ resource.Resource                 = (*PlaywrightCodeBundleResource)(nil)
	_ resource.ResourceWithConfigure    = (*PlaywrightCodeBundleResource)(nil)
	_ resource.ResourceWithUpgradeState = (*PlaywrightCodeBundleResource)(nil)
)

// NewPlaywrightCodeBundleResource returns a new checkly_playwright_code_bundle
// resource.
func NewPlaywrightCodeBundleResource() resource.Resource {
	return &PlaywrightCodeBundleResource{}
}

// PlaywrightCodeBundleResource implements the checkly_playwright_code_bundle
// resource.
type PlaywrightCodeBundleResource struct {
	client checkly.Client
}

// PlaywrightCodeBundleResourceModel describes the resource data model.
type PlaywrightCodeBundleResourceModel struct {
	ID              types.String `tfsdk:"id"`
	PrebuiltArchive types.List   `tfsdk:"prebuilt_archive"`
	Metadata        types.String `tfsdk:"metadata"`
}

// PlaywrightCodeBundlePrebuiltArchiveModel describes the prebuilt_archive
// block data model.
type PlaywrightCodeBundlePrebuiltArchiveModel struct {
	File types.String `tfsdk:"file"`
}

var playwrightCodeBundlePrebuiltArchiveAttrTypes = map[string]attr.Type{
	"file": types.StringType,
}

func (r *PlaywrightCodeBundleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playwright_code_bundle"
}

func (r *PlaywrightCodeBundleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     sdkv2SchemaVersion + 1,
		Description: "A managed code bundle which can be used in Playwright Check Suite resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the code bundle.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			playwrightMetadataAttributeName: schema.StringAttribute{
				Computed: true,
				Description: "An opaque blob of generated metadata. The " +
					"value is not intended to be user-consumable and should " +
					"be passed as-is to a Playwright check resource.",
				PlanModifiers: []planmodifier.String{
					playwrightCodeBundleMetadataPlanModifier{},
				},
			},
		},
		Blocks: map[string]schema.Block{
			playwrightPrebuiltArchiveAttributeName: schema.ListNestedBlock{
				Description: "A prebuilt archive containing the code bundle.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtMost(1),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"file": schema.StringAttribute{
							Required:    true,
							Description: "Path to the archive file.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								playwrightFileExistsValidator{},
								playwrightGzipArchiveValidator{},
							},
						},
					},
				},
			},
		},
	}
}

// UpgradeState upgrades state written by the provider's SDK v2
// implementation; see sdkv2StateUpgraders.
func (r *PlaywrightCodeBundleResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return sdkv2StateUpgraders(ctx, r)
}

func (r *PlaywrightCodeBundleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *PlaywrightCodeBundleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PlaywrightCodeBundleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	archive := playwrightCodeBundleArchiveFromList(ctx, plan.PrebuiltArchive, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if archive == nil {
		resp.Diagnostics.AddError(
			"Unable to Create Playwright Code Bundle",
			"Bundle has no source.",
		)
		return
	}

	// The metadata is normally computed at plan time by the plan modifier.
	// It can remain unknown when the archive file path itself was not known
	// during plan; in that case it is computed here instead.
	if plan.Metadata.IsUnknown() {
		if err := archive.InspectArchivePaths(); err != nil {
			resp.Diagnostics.AddError("Invalid Playwright Code Bundle Archive", err.Error())
			return
		}

		checksum, err := archive.ChecksumSha256()
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Playwright Code Bundle",
				fmt.Sprintf("Failed to calculate source archive checksum: %s", err),
			)
			return
		}

		metadata := playwrightCodeBundleComputeMetadata(archive, checksum, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		plan.Metadata = types.StringValue(metadata.EncodeToString())
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := archive.Upload(apiCtx, r.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Playwright Code Bundle",
			fmt.Sprintf("Failed to upload source archive: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(base64.StdEncoding.EncodeToString([]byte(result.Key)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PlaywrightCodeBundleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PlaywrightCodeBundleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := base64.StdEncoding.DecodeString(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Playwright Code Bundle",
			fmt.Sprintf("Invalid code bundle identifier %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	apiCtx, cancel := contextWithAPITimeout(ctx)
	defer cancel()

	result, err := r.client.PeekCodeBundle(apiCtx, string(key))
	if err != nil {
		if errors.Is(err, checkly.ErrCodeBundleNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to Read Playwright Code Bundle",
			fmt.Sprintf("Failed to peek code bundle: %s", err),
		)
		return
	}

	// The remote checksum is stored into the metadata so that a mismatch
	// with the local archive is detected at plan time (the metadata plan
	// modifier recomputes the metadata from the local file and requires
	// replacement when it differs).
	if result.ChecksumSha256 != "" {
		metadata, err := PlaywrightCodeBundleMetadataFromString(state.Metadata.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Playwright Code Bundle",
				fmt.Sprintf("Failed to decode code bundle metadata: %s", err),
			)
			return
		}

		metadata.ChecksumSha256 = result.ChecksumSha256

		state.Metadata = types.StringValue(metadata.EncodeToString())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PlaywrightCodeBundleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every attribute of the resource requires replacement when changed, so
	// an in-place update can never actually happen. The method only exists
	// to satisfy the resource.Resource interface.
	var plan PlaywrightCodeBundleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PlaywrightCodeBundleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The code bundle cannot actually be deleted. It will be cleaned up when
	// it is no longer in use.
}

var _ planmodifier.String = (*playwrightCodeBundleMetadataPlanModifier)(nil)

// playwrightCodeBundleMetadataPlanModifier computes the metadata attribute
// from the local archive file at plan time. Because the metadata is derived
// entirely from the local file, it can be resolved to a known value before
// apply, which lets dependent resources (Playwright check suites) use it for
// their own plan-time computations.
//
// When the computed metadata differs from the metadata stored in state, the
// resource requires replacement: the bundle contents changed and a new
// archive must be uploaded.
type playwrightCodeBundleMetadataPlanModifier struct{}

func (m playwrightCodeBundleMetadataPlanModifier) Description(ctx context.Context) string {
	return "Computes code bundle metadata from the local archive file and requires replacement when it changes."
}

func (m playwrightCodeBundleMetadataPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m playwrightCodeBundleMetadataPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var archiveList types.List

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, playwrightPrebuiltArchivePath(), &archiveList)...)
	if resp.Diagnostics.HasError() {
		return
	}

	archive := playwrightCodeBundleArchiveFromList(ctx, archiveList, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || archive == nil {
		return
	}

	// Run before the checksum comparison below, so that an archive which is
	// already in state gets validated too, not just one that changed.
	if err := archive.InspectArchivePaths(); err != nil {
		resp.Diagnostics.AddError("Invalid Playwright Code Bundle Archive", err.Error())
		return
	}

	checksum, err := archive.ChecksumSha256()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Inspect Playwright Code Bundle Archive",
			fmt.Sprintf("Failed to calculate source archive checksum: %s", err),
		)
		return
	}

	if !req.StateValue.IsNull() {
		prior, err := PlaywrightCodeBundleMetadataFromString(req.StateValue.ValueString())
		if err == nil &&
			prior.Version == PlaywrightCodeBundleMetadataCurrentVersion &&
			prior.ChecksumSha256 == checksum {
			// The stored metadata was produced by the same provider version
			// from an identical archive; it needs no update.
			resp.PlanValue = req.StateValue
			return
		}
	}

	metadata := playwrightCodeBundleComputeMetadata(archive, checksum, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	encoded := metadata.EncodeToString()

	resp.PlanValue = types.StringValue(encoded)

	if !req.StateValue.IsNull() && req.StateValue.ValueString() != encoded {
		resp.RequiresReplace = true
	}
}

// playwrightPrebuiltArchivePath returns the attribute path of the
// prebuilt_archive block.
func playwrightPrebuiltArchivePath() tfpath.Path {
	return tfpath.Root(playwrightPrebuiltArchiveAttributeName)
}

// playwrightCodeBundleArchiveFromList converts the prebuilt_archive block
// value to an archive helper. It returns nil when the block is absent or its
// file path is not yet known.
func playwrightCodeBundleArchiveFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) *PlaywrightCodeBundlePrebuiltArchiveAttribute {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var models []PlaywrightCodeBundlePrebuiltArchiveModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return nil
	}

	file := models[0].File
	if file.IsNull() || file.IsUnknown() {
		return nil
	}

	return &PlaywrightCodeBundlePrebuiltArchiveAttribute{
		File: file.ValueString(),
	}
}

// playwrightCodeBundleComputeMetadata inspects the archive and derives the
// full code bundle metadata from its contents.
func playwrightCodeBundleComputeMetadata(
	archive *PlaywrightCodeBundlePrebuiltArchiveAttribute,
	checksum string,
	diags *diag.Diagnostics,
) *PlaywrightCodeBundleMetadata {
	lockfileInfo, err := archive.InspectLockfile("@playwright/test", InspectLockfileOptions{
		PackageJSONExcludedFields: []string{
			// Exclude "version" because CI workflows often
			// stamp it with a commit hash or build number.
			// Including it would invalidate the dependency
			// cache on every build even when no dependencies
			// actually changed.
			"version",
		},
	})
	if err != nil {
		diags.AddError(
			"Unable to Inspect Playwright Code Bundle Archive",
			fmt.Sprintf("Failed to inspect lockfile in archive: %s", err),
		)
		return nil
	}

	if lockfileInfo == nil {
		diags.AddError(
			"Unable to Inspect Playwright Code Bundle Archive",
			"no lockfile found at the root of the archive; "+
				"the archive must contain a package-lock.json, pnpm-lock.yaml, yarn.lock, or bun.lock at the root level.",
		)
		return nil
	}

	if lockfileInfo.PackageVersion == "" {
		diags.AddError(
			"Unable to Inspect Playwright Code Bundle Archive",
			"the lockfile does not contain @playwright/test; "+
				"add @playwright/test to the project's dependencies and regenerate the lockfile.",
		)
		return nil
	}

	workingDir, err := archive.DetectWorkingDir()
	if err != nil {
		diags.AddError(
			"Unable to Inspect Playwright Code Bundle Archive",
			fmt.Sprintf("Failed to detect working directory in archive: %s", err),
		)
		return nil
	}

	return &PlaywrightCodeBundleMetadata{
		Version:           PlaywrightCodeBundleMetadataCurrentVersion,
		ChecksumSha256:    checksum,
		PlaywrightVersion: lockfileInfo.PackageVersion,
		PackageManager:    lockfileInfo.PackageManager,
		CacheHash:         lockfileInfo.ChecksumSha256,
		WorkingDir:        workingDir,
		Engine:            lockfileInfo.Engine,
		EngineVersion:     lockfileInfo.EngineVersion,
		EngineRawVersion:  lockfileInfo.EngineRawVersion,
		EngineSource:      lockfileInfo.EngineSource,
	}
}

const PlaywrightCodeBundleMetadataCurrentVersion = 7

type PlaywrightCodeBundleMetadata struct {
	Version           int    `json:"v"`
	ChecksumSha256    string `json:"s256"`
	PlaywrightVersion string `json:"pwv,omitempty"`
	PackageManager    string `json:"pm,omitempty"`
	CacheHash         string `json:"ch,omitempty"`
	WorkingDir        string `json:"wd,omitempty"`
	Engine            string `json:"en,omitempty"`
	EngineVersion     string `json:"ev,omitempty"`
	EngineRawVersion  string `json:"erv,omitempty"`
	EngineSource      string `json:"es,omitempty"`
}

func PlaywrightCodeBundleMetadataFromString(s string) (*PlaywrightCodeBundleMetadata, error) {
	if s == "" {
		return new(PlaywrightCodeBundleMetadata), nil
	}

	b64, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("failed to decode code bundle metadata %q: %w", s, err)
	}

	dec := json.NewDecoder(bytes.NewReader(b64))

	var t PlaywrightCodeBundleMetadata

	err = dec.Decode(&t)
	if err != nil {
		return nil, fmt.Errorf("failed to decode code bundle metadata %q: %w", s, err)
	}

	return &t, err
}

func (t *PlaywrightCodeBundleMetadata) EncodeToString() string {
	buf := new(bytes.Buffer)

	enc := json.NewEncoder(buf)

	err := enc.Encode(t)
	if err != nil {
		// This should not be possible, so let's panic.
		panic(fmt.Errorf("failed to encode code bundle data %+v: %w", t, err))
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// PlaywrightCodeBundlePrebuiltArchiveAttribute provides access to a local
// prebuilt archive file.
type PlaywrightCodeBundlePrebuiltArchiveAttribute struct {
	File string
}

func (a *PlaywrightCodeBundlePrebuiltArchiveAttribute) ChecksumSha256() (string, error) {
	file, err := os.Open(a.File)
	if err != nil {
		return "", fmt.Errorf("failed to open archive file %q: %w", a.File, err)
	}
	defer file.Close()

	checksum := checksumSha256(file)

	return checksum, nil
}

// LockfileInfo contains information extracted from a lockfile found in an archive.
type LockfileInfo struct {
	PackageManager   string
	PackageVersion   string
	ChecksumSha256   string
	Engine           string
	EngineVersion    string
	EngineRawVersion string
	EngineSource     string
}

type lockfileParser struct {
	packageManager string
	parse          func(io.Reader, string) (string, error)
}

var lockfileParsers = map[string]lockfileParser{
	"package-lock.json": {"npm", extractPackageVersionFromPackageLock},
	"pnpm-lock.yaml":    {"pnpm", extractPackageVersionFromPnpmLock},
	"yarn.lock":         {"yarn", extractPackageVersionFromYarnLock},
	"bun.lock":          {"bun", extractPackageVersionFromBunLock},
}

// ErrUnsupportedBunLockb signals that the archive contains bun.lockb (Bun's
// legacy binary lockfile format) but no parseable text-based lockfile.
var ErrUnsupportedBunLockb = errors.New(
	"the archive contains a bun.lockb binary lockfile, which is not supported; " +
		"regenerate it as text with `bun install --save-text-lockfile` or set " +
		"`saveTextLockfile = true` under `[install.lockfile]` in bunfig.toml, then rebuild the archive",
)

// InspectLockfileOptions controls optional behavior of InspectLockfile.
type InspectLockfileOptions struct {
	// PackageJSONExcludedFields lists top-level keys to remove from every
	// package.json before it contributes to ChecksumSha256. Useful for
	// fields that don't affect runtime behavior, like "version".
	PackageJSONExcludedFields []string
}

type packageJSONEntry struct {
	path string
	raw  []byte
}

type npmrcEntry struct {
	path string
	raw  []byte
}

// InspectLockfile opens the tar.gz archive and searches for a lockfile
// (package-lock.json, pnpm-lock.yaml, yarn.lock, or bun.lock) at the root
// of the archive. If found, it returns the detected package manager and
// the resolved version of the given package.
//
// ChecksumSha256 covers the lockfile contents, every package.json outside
// node_modules (at any depth), and every .npmrc outside node_modules. Each
// package.json is canonicalized as JSON with opts.PackageJSONExcludedFields
// removed from the top level, so cosmetic changes and excluded fields don't
// influence the checksum. .npmrc files contribute the raw SHA-256 of their
// bytes, so any change to registry configuration invalidates the checksum.
func (a *PlaywrightCodeBundlePrebuiltArchiveAttribute) InspectLockfile(
	packageName string,
	opts InspectLockfileOptions,
) (*LockfileInfo, error) {
	file, err := os.Open(a.File)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive file %q: %w", a.File, err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader for %q: %w", a.File, err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var (
		sawBunLockb    bool
		lockfileName   string
		lockfileHash   []byte
		packageManager string
		packageVersion string
		packageJSONs   []packageJSONEntry
		npmrcs         []npmrcEntry
		engineFiles    = make(map[string][]byte)
	)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read archive %q: %w", a.File, err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		name := strings.TrimPrefix(header.Name, "./")

		if path.Base(name) == "package.json" && !hasNodeModulesSegment(name) {
			raw, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read %q from archive %q: %w", header.Name, a.File, err)
			}
			packageJSONs = append(packageJSONs, packageJSONEntry{path: name, raw: raw})
			continue
		}

		if path.Base(name) == ".npmrc" && !hasNodeModulesSegment(name) {
			raw, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read %q from archive %q: %w", header.Name, a.File, err)
			}
			npmrcs = append(npmrcs, npmrcEntry{path: name, raw: raw})
			continue
		}

		// Only consider lockfiles and engine version files at the root of the archive.
		if strings.Contains(name, "/") {
			continue
		}

		switch name {
		case ".node-version", ".nvmrc", ".tool-versions", ".bun-version":
			raw, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read %q from archive %q: %w", header.Name, a.File, err)
			}
			engineFiles[name] = raw
			continue
		}

		if name == "bun.lockb" {
			sawBunLockb = true
			continue
		}

		parser, ok := lockfileParsers[name]
		if !ok {
			continue
		}

		if lockfileName != "" {
			// Already parsed a lockfile; ignore any duplicates.
			continue
		}

		// Hash the lockfile content as it flows through the parser.
		hash := sha256.New()
		tee := io.TeeReader(tr, hash)

		version, err := parser.parse(tee, packageName)
		if err != nil {
			return nil, fmt.Errorf("failed to extract package version from %q in archive: %w", header.Name, err)
		}

		// Drain any remaining bytes the parser didn't consume so
		// the checksum covers the entire lockfile.
		if _, err := io.Copy(io.Discard, tee); err != nil {
			return nil, fmt.Errorf("failed to read lockfile %q from archive: %w", header.Name, err)
		}

		lockfileName = name
		lockfileHash = hash.Sum(nil)
		packageManager = parser.packageManager
		packageVersion = version
	}

	if lockfileName == "" {
		if sawBunLockb {
			return nil, ErrUnsupportedBunLockb
		}
		return nil, nil
	}

	checksum, err := composeBundleChecksum(lockfileName, lockfileHash, packageJSONs, npmrcs, opts.PackageJSONExcludedFields)
	if err != nil {
		return nil, fmt.Errorf("failed to compute archive checksum: %w", err)
	}

	// Add root package.json to engine files so the detector can read the
	// Volta pin (volta.node) and the engines field.
	for _, entry := range packageJSONs {
		if entry.path == "package.json" {
			engineFiles["package.json"] = entry.raw
			break
		}
	}

	engineResult := detectEngine(engineFiles, packageManager)

	info := &LockfileInfo{
		PackageManager: packageManager,
		PackageVersion: packageVersion,
		ChecksumSha256: checksum,
	}
	if engineResult != nil && engineResult.Engine != nil {
		info.Engine = engineResult.Engine.Name
		info.EngineVersion = engineResult.Engine.Version
		info.EngineRawVersion = engineResult.RawVersion
		info.EngineSource = engineResult.Source
	}
	return info, nil
}

func hasNodeModulesSegment(p string) bool {
	return strings.Contains("/"+p+"/", "/node_modules/")
}

// canonicalizePackageJSON parses raw as JSON, deletes the named top-level
// fields, and re-encodes. Re-encoding via json.Marshal produces output with
// map keys sorted alphabetically, so whitespace and key order in the source
// don't affect the result.
func canonicalizePackageJSON(raw []byte, excludedFields []string) ([]byte, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}
	for _, f := range excludedFields {
		delete(obj, f)
	}
	return json.Marshal(obj)
}

// composeBundleChecksum combines the lockfile hash, every canonicalized
// package.json, and every .npmrc (each set sorted by path) into a single
// SHA-256. Records are length-prefixed to prevent collisions from ambiguous
// concatenation.
//
// This must stay byte-for-byte compatible with the Checkly CLI's
// composeCacheHash (packages/cli/src/services/check-parser/cache-hash.ts):
// same record framing, same label prefixes, and the same record order —
// lockfile, then package.json records (sorted), then npmrc records (sorted).
// An .npmrc record's content is the raw SHA-256 digest of the file bytes
// (not canonicalized, unlike package.json), matching the CLI.
func composeBundleChecksum(
	lockfileName string,
	lockfileHash []byte,
	packageJSONs []packageJSONEntry,
	npmrcs []npmrcEntry,
	excludedFields []string,
) (string, error) {
	sort.Slice(packageJSONs, func(i, j int) bool {
		return packageJSONs[i].path < packageJSONs[j].path
	})
	sort.Slice(npmrcs, func(i, j int) bool {
		return npmrcs[i].path < npmrcs[j].path
	})

	h := sha256.New()
	writeRecord := func(label string, content []byte) {
		var lenBuf [8]byte
		binary.BigEndian.PutUint64(lenBuf[:], uint64(len(label)))
		h.Write(lenBuf[:])
		h.Write([]byte(label))
		binary.BigEndian.PutUint64(lenBuf[:], uint64(len(content)))
		h.Write(lenBuf[:])
		h.Write(content)
	}

	writeRecord("lockfile:"+lockfileName, lockfileHash)

	for _, entry := range packageJSONs {
		canonical, err := canonicalizePackageJSON(entry.raw, excludedFields)
		if err != nil {
			return "", fmt.Errorf("failed to canonicalize %q: %w", entry.path, err)
		}
		writeRecord("package.json:"+entry.path, canonical)
	}

	for _, entry := range npmrcs {
		sum := sha256.Sum256(entry.raw)
		writeRecord("npmrc:"+entry.path, sum[:])
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

var playwrightConfigExtensions = map[string]bool{
	".ts": true, ".mts": true, ".cts": true,
	".js": true, ".mjs": true, ".cjs": true,
}

// isPlaywrightConfig returns true if the filename matches
// playwright*.config.{ts,mts,cts,js,mjs,cjs}.
func isPlaywrightConfig(name string) bool {
	base := path.Base(name)
	if !strings.HasPrefix(base, "playwright") {
		return false
	}

	// Find ".config." after the "playwright" prefix.
	rest := base[len("playwright"):]
	idx := strings.Index(rest, ".config.")
	if idx < 0 {
		return false
	}

	ext := rest[idx+len(".config.")-1:] // includes the leading dot
	return playwrightConfigExtensions[ext]
}

// DetectWorkingDir scans the archive for a Playwright config file and
// returns the directory of the closest ancestor package.json. If the
// config is at the root or no config is found, it returns an empty string.
func (a *PlaywrightCodeBundlePrebuiltArchiveAttribute) DetectWorkingDir() (string, error) {
	file, err := os.Open(a.File)
	if err != nil {
		return "", fmt.Errorf("failed to open archive file %q: %w", a.File, err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader for %q: %w", a.File, err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var configPaths []string
	packageJSONDirs := map[string]bool{}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read archive %q: %w", a.File, err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		name := strings.TrimPrefix(header.Name, "./")

		if isPlaywrightConfig(name) {
			configPaths = append(configPaths, name)
		}

		if path.Base(name) == "package.json" {
			packageJSONDirs[path.Dir(name)] = true
		}
	}

	if len(configPaths) == 0 {
		return ".", nil
	}

	// Pick the config with the shortest path. Break ties lexicographically.
	shortest := configPaths[0]
	for _, p := range configPaths[1:] {
		if len(p) < len(shortest) || (len(p) == len(shortest) && p < shortest) {
			shortest = p
		}
	}

	// Walk up from the config's directory to find the closest package.json.
	dir := path.Dir(shortest)
	for {
		if packageJSONDirs[dir] {
			return dir, nil
		}

		parent := path.Dir(dir)
		if parent == dir {
			// Reached root without finding package.json.
			return ".", nil
		}
		dir = parent
	}
}

func (a *PlaywrightCodeBundlePrebuiltArchiveAttribute) Upload(
	ctx context.Context,
	client checkly.Client,
) (*checkly.CodeBundle, error) {
	stat, err := os.Stat(a.File)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source archive file %q does not exist", a.File)
		}

		return nil, fmt.Errorf("failed to stat archive file %q: %w", a.File, err)
	}

	checksum, err := a.ChecksumSha256()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum for archive file %q: %w", a.File, err)
	}

	file, err := os.Open(a.File)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive file %q: %w", a.File, err)
	}

	codeBundle, err := client.UploadCodeBundle(ctx, file, stat.Size(), checkly.UploadCodeBundleOptions{
		ChecksumSha256: checksum,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload code bundle %q: %w", a.File, err)
	}

	return codeBundle, nil
}
