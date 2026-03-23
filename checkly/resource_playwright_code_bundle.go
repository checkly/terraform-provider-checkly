package checkly

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
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
							ValidateFunc: validateFileExists(),
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
					case bundle.Data.Version < 2:
						// Data should be updated.
					case checksum != bundle.Data.ChecksumSha256:
						// Data should be updated.
					default:
						// Data needs no update.
						return nil
					}

					lockfileInfo, err := bundle.PrebuiltArchive.InspectLockfile("@playwright/test")
					if err != nil {
						return fmt.Errorf("failed to inspect lockfile in archive: %v", err)
					}

					bundle.Data.Version = 2
					bundle.Data.ChecksumSha256 = checksum
					if lockfileInfo != nil {
						bundle.Data.PlaywrightVersion = lockfileInfo.PackageVersion
						bundle.Data.PackageManager = lockfileInfo.PackageManager
						bundle.Data.LockfileChecksum = lockfileInfo.ChecksumSha256
					}

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
	Version            int    `json:"v"`
	ChecksumSha256     string `json:"s256"`
	PlaywrightVersion  string `json:"pwv,omitempty"`
	PackageManager     string `json:"pm,omitempty"`
	LockfileChecksum   string `json:"lcs,omitempty"`
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
	PackageManager   string
	PackageVersion   string
	ChecksumSha256   string
}

type lockfileParser struct {
	packageManager string
	parse          func(io.Reader, string) (string, error)
}

var lockfileParsers = map[string]lockfileParser{
	"package-lock.json": {"npm", extractPackageVersionFromPackageLock},
	"pnpm-lock.yaml":    {"pnpm", extractPackageVersionFromPnpmLock},
	"yarn.lock":         {"yarn", extractPackageVersionFromYarnLock},
}

// InspectLockfile opens the tar.gz archive and searches for a lockfile
// (package-lock.json, pnpm-lock.yaml, or yarn.lock). If found, it returns
// the detected package manager and the resolved version of the given package.
func (a *PlaywrightCodeBundlePrebuiltArchiveAttribute) InspectLockfile(packageName string) (*LockfileInfo, error) {
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

		// Skip files inside node_modules
		if strings.Contains(header.Name, "node_modules/") {
			continue
		}

		base := path.Base(header.Name)
		parser, ok := lockfileParsers[base]
		if !ok {
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

		return &LockfileInfo{
			PackageManager: parser.packageManager,
			PackageVersion: version,
			ChecksumSha256: hex.EncodeToString(hash.Sum(nil)),
		}, nil
	}

	return nil, nil
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
