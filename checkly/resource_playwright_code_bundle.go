package checkly

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

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	prebuiltArchiveAttributeName = "prebuilt_archive"
	metadataAttributeName        = "metadata"
)

func resourcePlaywrightCodeBundle() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePlaywrightCodeBundleCreate,
		ReadContext:   resourcePlaywrightCodeBundleRead,
		DeleteContext: resourcePlaywrightCodeBundleDelete,
		Description:   "A managed code bundle which can be used in Playwright Check Suite resources.",
		Schema: map[string]*schema.Schema{
			prebuiltArchiveAttributeName: {
				Description: "A prebuilt archive containing the code bundle.",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"file": {
							Description:  "Path to the archive file.",
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateAll(validateFileExists(), validateGzipArchive()),
						},
					},
				},
			},
			metadataAttributeName: {
				Description: "An opaque blob of generated metadata. The " +
					"value is not intended to be user-consumable and should " +
					"be passed as-is to a Playwright check resource.",
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
		},
		CustomizeDiff: customdiff.Sequence(
			func(_ context.Context, diff *schema.ResourceDiff, meta any) error {
				bundle, err := PlaywrightCodeBundleResourceFromResourceDiff(diff)
				if err != nil {
					return fmt.Errorf("failed to thaw code bundle from resource diff: %v", err)
				}

				switch {
				case bundle.PrebuiltArchive != nil:
					checksum, err := bundle.PrebuiltArchive.ChecksumSha256()
					if err != nil {
						return fmt.Errorf("failed to calculate source archive checksum: %v", err)
					}

					switch {
					case bundle.Data.Version < 4:
						// Data should be updated.
					case checksum != bundle.Data.ChecksumSha256:
						// Data should be updated.
					default:
						// Data needs no update.
						return nil
					}

					lockfileInfo, err := bundle.PrebuiltArchive.InspectLockfile("@playwright/test", InspectLockfileOptions{
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
						return fmt.Errorf("failed to inspect lockfile in archive: %w", err)
					}

					if lockfileInfo == nil {
						return fmt.Errorf(
							"no lockfile found at the root of the archive; " +
								"the archive must contain a package-lock.json, pnpm-lock.yaml, yarn.lock, or bun.lock at the root level",
						)
					}

					if lockfileInfo.PackageVersion == "" {
						return fmt.Errorf(
							"the lockfile does not contain @playwright/test; " +
								"add @playwright/test to the project's dependencies and regenerate the lockfile",
						)
					}

					workingDir, err := bundle.PrebuiltArchive.DetectWorkingDir()
					if err != nil {
						return fmt.Errorf("failed to detect working directory in archive: %v", err)
					}

					bundle.Data.Version = 4
					bundle.Data.ChecksumSha256 = checksum
					bundle.Data.PlaywrightVersion = lockfileInfo.PackageVersion
					bundle.Data.PackageManager = lockfileInfo.PackageManager
					bundle.Data.CacheHash = lockfileInfo.ChecksumSha256
					bundle.Data.WorkingDir = workingDir

					err = diff.SetNew(metadataAttributeName, bundle.Data.EncodeToString())
					if err != nil {
						return fmt.Errorf("failed to set %q: %v", metadataAttributeName, err)
					}

					return nil
				default:
					return fmt.Errorf("bundle has no source")
				}
			},
		),
	}
}

func resourcePlaywrightCodeBundleCreate(
	ctx context.Context,
	d *schema.ResourceData,
	client any,
) (diags diag.Diagnostics) {
	ctx, cancel := context.WithTimeout(ctx, apiCallTimeout())
	defer cancel()

	bundle, err := PlaywrightCodeBundleResourceFromResourceData(d)
	if err != nil {
		return diag.Errorf("failed to thaw code bundle from resource data: %v", err)
	}

	switch {
	case bundle.PrebuiltArchive != nil:
		result, err := bundle.PrebuiltArchive.Upload(ctx, client.(checkly.Client))
		if err != nil {
			return diag.Errorf("failed to upload source archive: %v", err)
		}

		d.SetId(base64.StdEncoding.EncodeToString([]byte(result.Key)))

		err = d.Set(prebuiltArchiveAttributeName, bundle.PrebuiltArchive.ToList())
		if err != nil {
			return diag.Errorf("failed to set %q state: %v", prebuiltArchiveAttributeName, err)
		}

		err = d.Set(metadataAttributeName, bundle.Data.EncodeToString())
		if err != nil {
			return diag.Errorf("failed to set %q state: %v", metadataAttributeName, err)
		}

		return nil
	default:
		return diag.Errorf("bundle has no source")
	}
}

func resourcePlaywrightCodeBundleRead(
	ctx context.Context,
	d *schema.ResourceData,
	client any,
) (diags diag.Diagnostics) {
	ctx, cancel := context.WithTimeout(ctx, apiCallTimeout())
	defer cancel()

	key, err := base64.StdEncoding.DecodeString(d.Id())
	if err != nil {
		return diag.Errorf("failed to thaw code bundle from resource data: %v", err)
	}

	bundle, err := PlaywrightCodeBundleResourceFromResourceData(d)
	if err != nil {
		return diag.Errorf("failed to thaw code bundle from resource data: %v", err)
	}

	result, err := client.(checkly.Client).PeekCodeBundle(ctx, string(key))
	if err != nil {
		if errors.Is(err, checkly.ErrCodeBundleNotFound) {
			d.SetId("")
			return nil
		}

		return diag.Errorf("failed to peek code bundle: %v", err)
	}

	if result.ChecksumSha256 != "" {
		bundle.Data.ChecksumSha256 = result.ChecksumSha256

		err = d.Set(metadataAttributeName, bundle.Data.EncodeToString())
		if err != nil {
			return diag.Errorf("failed to set %q state: %v", metadataAttributeName, err)
		}
	}

	return nil
}

func resourcePlaywrightCodeBundleDelete(
	ctx context.Context,
	d *schema.ResourceData,
	client any,
) (diags diag.Diagnostics) {
	// The code bundle cannot actually be deleted. It will be cleaned up when
	// it is no longer in use.
	return diags
}

type PlaywrightCodeBundleMetadata struct {
	Version           int    `json:"v"`
	ChecksumSha256    string `json:"s256"`
	PlaywrightVersion string `json:"pwv,omitempty"`
	PackageManager    string `json:"pm,omitempty"`
	CacheHash         string `json:"ch,omitempty"`
	WorkingDir        string `json:"wd,omitempty"`
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
		panic(fmt.Errorf("failed to encode code bundle data %q: %w", t, err))
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

type PlaywrightCodeBundleResource struct {
	ID              string
	Data            *PlaywrightCodeBundleMetadata
	PrebuiltArchive *PlaywrightCodeBundlePrebuiltArchiveAttribute
}

func PlaywrightCodeBundleResourceFromResourceData(
	d *schema.ResourceData,
) (PlaywrightCodeBundleResource, error) {
	prebuiltArchiveAttr, err := PlaywrightCodeBundlePrebuiltArchiveAttributeFromList(d.Get(prebuiltArchiveAttributeName).([]any))
	if err != nil {
		return PlaywrightCodeBundleResource{}, err
	}

	data, err := PlaywrightCodeBundleMetadataFromString(d.Get(metadataAttributeName).(string))
	if err != nil {
		return PlaywrightCodeBundleResource{}, err
	}

	resource := PlaywrightCodeBundleResource{
		ID:              d.Id(),
		Data:            data,
		PrebuiltArchive: prebuiltArchiveAttr,
	}

	return resource, nil
}

func PlaywrightCodeBundleResourceFromResourceDiff(
	d *schema.ResourceDiff,
) (PlaywrightCodeBundleResource, error) {
	prebuiltArchiveAttr, err := PlaywrightCodeBundlePrebuiltArchiveAttributeFromList(d.Get(prebuiltArchiveAttributeName).([]any))
	if err != nil {
		return PlaywrightCodeBundleResource{}, err
	}

	data, err := PlaywrightCodeBundleMetadataFromString(d.Get(metadataAttributeName).(string))
	if err != nil {
		return PlaywrightCodeBundleResource{}, err
	}

	resource := PlaywrightCodeBundleResource{
		ID:              d.Id(),
		Data:            data,
		PrebuiltArchive: prebuiltArchiveAttr,
	}

	return resource, nil
}

type PlaywrightCodeBundlePrebuiltArchiveAttribute struct {
	File string
}

func PlaywrightCodeBundlePrebuiltArchiveAttributeFromList(
	list []any,
) (*PlaywrightCodeBundlePrebuiltArchiveAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := PlaywrightCodeBundlePrebuiltArchiveAttribute{
		File: m["file"].(string),
	}

	return &a, nil
}

func (a *PlaywrightCodeBundlePrebuiltArchiveAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"file": a.File,
		},
	}
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
	PackageManager string
	PackageVersion string
	ChecksumSha256 string
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

// InspectLockfile opens the tar.gz archive and searches for a lockfile
// (package-lock.json, pnpm-lock.yaml, yarn.lock, or bun.lock) at the root
// of the archive. If found, it returns the detected package manager and
// the resolved version of the given package.
//
// ChecksumSha256 covers both the lockfile contents and every package.json
// outside node_modules (at any depth). Each package.json is canonicalized
// as JSON with opts.PackageJSONExcludedFields removed from the top level,
// so cosmetic changes and excluded fields don't influence the checksum.
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

		// Only consider lockfiles at the root of the archive.
		if strings.Contains(name, "/") {
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

	checksum, err := composeBundleChecksum(lockfileName, lockfileHash, packageJSONs, opts.PackageJSONExcludedFields)
	if err != nil {
		return nil, fmt.Errorf("failed to compute archive checksum: %w", err)
	}

	return &LockfileInfo{
		PackageManager: packageManager,
		PackageVersion: packageVersion,
		ChecksumSha256: checksum,
	}, nil
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

// composeBundleChecksum combines the lockfile hash and every canonicalized
// package.json (sorted by path) into a single SHA-256. Records are
// length-prefixed to prevent collisions from ambiguous concatenation.
func composeBundleChecksum(
	lockfileName string,
	lockfileHash []byte,
	packageJSONs []packageJSONEntry,
	excludedFields []string,
) (string, error) {
	sort.Slice(packageJSONs, func(i, j int) bool {
		return packageJSONs[i].path < packageJSONs[j].path
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
