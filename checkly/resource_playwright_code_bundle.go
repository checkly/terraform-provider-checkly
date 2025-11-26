package checkly

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	checkly "github.com/checkly/checkly-go-sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePlaywrightCodeBundle() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePlaywrightCodeBundleCreate,
		ReadContext:   resourcePlaywrightCodeBundleRead,
		DeleteContext: resourcePlaywrightCodeBundleDelete,
		Description:   "A managed code bundle which can be used in Playwright Check Suite resources.",
		Schema: map[string]*schema.Schema{
			"source_archive": {
				Description: "",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"file": {
							Description:  "",
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateFileExists(),
						},
					},
				},
			},
			"data": {
				Description: "An opaque, computed value containing auxiliary " +
					"data of the code bundle. This value should be passed " +
					"as-is to a check resource.",
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
				case bundle.SourceArchive != nil:
					checksum, err := bundle.SourceArchive.ChecksumSha256()
					if err != nil {
						return fmt.Errorf("failed to calculate source archive checksum: %v", err)
					}

					switch {
					case bundle.Data.Version < 1:
						// Data should be updated.
					case checksum != bundle.Data.ChecksumSha256:
						// Data should be updated.
					default:
						// Data needs no update.
						return nil
					}

					bundle.Data.Version = 1
					bundle.Data.ChecksumSha256 = checksum

					err = diff.SetNew("data", bundle.Data.EncodeToString())
					if err != nil {
						return fmt.Errorf("failed to set %q: %v", "data", err)
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
	case bundle.SourceArchive != nil:
		result, err := bundle.SourceArchive.Upload(ctx, client.(checkly.Client))
		if err != nil {
			return diag.Errorf("failed to upload source archive: %v", err)
		}

		d.SetId(base64.StdEncoding.EncodeToString([]byte(result.Key)))

		err = d.Set("source_archive", bundle.SourceArchive.ToList())
		if err != nil {
			return diag.Errorf("failed to set %q state: %v", "source_archive", err)
		}

		err = d.Set("data", bundle.Data.EncodeToString())
		if err != nil {
			return diag.Errorf("failed to set %q state: %v", "data", err)
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

		err = d.Set("data", bundle.Data.EncodeToString())
		if err != nil {
			return diag.Errorf("failed to set %q state: %v", "data", err)
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

type PlaywrightCodeBundleData struct {
	Version        int    `json:"v"`
	ChecksumSha256 string `json:"s256"`
}

func PlaywrightCodeBundleDataFromString(s string) (*PlaywrightCodeBundleData, error) {
	if s == "" {
		return new(PlaywrightCodeBundleData), nil
	}

	b64, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("failed to decode code bundle data %q: %w", s, err)
	}

	dec := json.NewDecoder(bytes.NewReader(b64))

	var t PlaywrightCodeBundleData

	err = dec.Decode(&t)
	if err != nil {
		return nil, fmt.Errorf("failed to decode code bundle data %q: %w", s, err)
	}

	return &t, err
}

func (t *PlaywrightCodeBundleData) EncodeToString() string {
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
	ID            string
	Data          *PlaywrightCodeBundleData
	SourceArchive *PlaywrightCodeBundleSourceArchiveAttribute
}

func PlaywrightCodeBundleResourceFromResourceData(
	d *schema.ResourceData,
) (PlaywrightCodeBundleResource, error) {
	sourceArchiveAttr, err := PlaywrightCodeBundleSourceArchiveAttributeFromList(d.Get("source_archive").([]any))
	if err != nil {
		return PlaywrightCodeBundleResource{}, err
	}

	data, err := PlaywrightCodeBundleDataFromString(d.Get("data").(string))
	if err != nil {
		return PlaywrightCodeBundleResource{}, err
	}

	resource := PlaywrightCodeBundleResource{
		ID:            d.Id(),
		Data:          data,
		SourceArchive: sourceArchiveAttr,
	}

	return resource, nil
}

func PlaywrightCodeBundleResourceFromResourceDiff(
	d *schema.ResourceDiff,
) (PlaywrightCodeBundleResource, error) {
	sourceArchiveAttr, err := PlaywrightCodeBundleSourceArchiveAttributeFromList(d.Get("source_archive").([]any))
	if err != nil {
		return PlaywrightCodeBundleResource{}, err
	}

	data, err := PlaywrightCodeBundleDataFromString(d.Get("data").(string))
	if err != nil {
		return PlaywrightCodeBundleResource{}, err
	}

	resource := PlaywrightCodeBundleResource{
		ID:            d.Id(),
		Data:          data,
		SourceArchive: sourceArchiveAttr,
	}

	return resource, nil
}

type PlaywrightCodeBundleSourceArchiveAttribute struct {
	File string
}

func PlaywrightCodeBundleSourceArchiveAttributeFromList(
	list []any,
) (*PlaywrightCodeBundleSourceArchiveAttribute, error) {
	if len(list) == 0 {
		return nil, nil
	}

	m := list[0].(tfMap)

	a := PlaywrightCodeBundleSourceArchiveAttribute{
		File: m["file"].(string),
	}

	return &a, nil
}

func (a *PlaywrightCodeBundleSourceArchiveAttribute) ToList() []tfMap {
	if a == nil {
		return []tfMap{}
	}

	return []tfMap{
		{
			"file": a.File,
		},
	}
}

func (a *PlaywrightCodeBundleSourceArchiveAttribute) ChecksumSha256() (string, error) {
	file, err := os.Open(a.File)
	if err != nil {
		return "", fmt.Errorf("failed to open archive file %q: %w", a.File, err)
	}
	defer file.Close()

	checksum := checksumSha256(file)

	return checksum, nil
}

func (a *PlaywrightCodeBundleSourceArchiveAttribute) Upload(
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
