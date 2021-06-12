package provider

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	log "github.com/sirupsen/logrus"
)

// Terraform gets upset if the same diff run multiple times yields any differences. This
// regexp helps to replace the variable parts with fixed placeholders.
var sanitizationRegexp = regexp.MustCompile(`(\s+)(creationTimestamp|uid)[:]([^\n]+)`)

// profileResource defines a new kubeapply_profile resource instance. The only required field
// is a path to the manifests
func profileResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProfileCreate,
		ReadContext:   resourceProfileRead,
		UpdateContext: resourceProfileUpdate,
		DeleteContext: resourceProfileDelete,
		CustomizeDiff: resourceProfileCustomDiff,
		Schema: map[string]*schema.Schema{
			// Inputs
			"source": {
				Type:        schema.TypeString,
				Description: "Source for profile manifest files in local file system or remote git repo",
				Required:    true,
			},
			"parameters": {
				Type:        schema.TypeMap,
				Description: "Arbitrary parameters that will be used for profile expansion",
				Optional:    true,
			},
			"set": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Custom, JSON-encoded parameters to be merged parameters above",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"force_diff": {
				Type:        schema.TypeBool,
				Description: "Force a full diff even if no inputs changed",
				Optional:    true,
			},

			// Computed fields
			"diff": {
				Type:        schema.TypeMap,
				Description: "Diff result from applying changed files",
				Computed:    true,
			},
			"resources": {
				Type:        schema.TypeMap,
				Description: "Resources in this profile",
				Computed:    true,
			},
		},
	}
}

func resourceProfileCreate(
	ctx context.Context,
	data *schema.ResourceData,
	provider interface{},
) diag.Diagnostics {
	log.Infof(
		"Running create for source %s",
		data.Get("source").(string),
	)
	providerCtx := provider.(*providerContext)
	var diags diag.Diagnostics

	expandResult, err := providerCtx.expand(ctx, data)
	if err != nil {
		return diag.FromErr(err)
	}
	defer providerCtx.cleanExpanded(expandResult)

	results, err := providerCtx.apply(ctx, expandResult.expandedDir)
	log.Infof(
		"Apply results for source %s (err=%+v): %s",
		data.Get("source").(string),
		err,
		string(results),
	)
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  err.Error(),
				Detail:   string(results),
			},
		}
	}

	err = data.Set(
		"resources", expandResult.resources,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	// Null out diff so it's not persisted and we get a clean diff for the next apply.
	err = data.Set("diff", map[string]interface{}{})
	if err != nil {
		return diag.FromErr(err)
	}

	// Just make up an id from the timestamp
	data.SetId(fmt.Sprintf("%d", time.Now().UnixNano()))

	log.Infof(
		"Create successful for source %s",
		data.Get("source").(string),
	)
	return diags
}

func resourceProfileRead(
	ctx context.Context,
	data *schema.ResourceData,
	provider interface{},
) diag.Diagnostics {
	log.Infof(
		"Running read for source %s",
		data.Get("source").(string),
	)
	var diags diag.Diagnostics

	// There's nothing to do here since we only read the remote kube state if we're doing a
	// diff.
	return diags
}

func resourceProfileCustomDiff(
	ctx context.Context,
	data *schema.ResourceDiff,
	provider interface{},
) error {
	log.Infof(
		"Running custom diff for source %s",
		data.Get("source").(string),
	)
	providerCtx := provider.(*providerContext)
	expandResult, err := providerCtx.expand(ctx, data)
	if err != nil {
		return err
	}
	defer providerCtx.cleanExpanded(expandResult)

	log.Infof(
		"Found %d manifests with overall hash of %s for source %s",
		len(expandResult.manifests),
		expandResult.totalHash,
		data.Get("source").(string),
	)

	// Set resources
	if err := data.SetNew(
		"resources",
		expandResult.resources,
	); err != nil {
		return err
	}

	if data.HasChange("resources") || data.Get("force_diff").(bool) {
		log.Infof(
			"Resources have changed for source %s",
			data.Get("source").(string),
		)
		var results map[string]interface{}

		if err := providerCtx.createNamespaces(ctx, expandResult.manifests); err != nil {
			return err
		}

		diffs, err := providerCtx.diff(ctx, expandResult.expandedDir)
		if err != nil {
			return err
		}
		log.Infof(
			"Got structured diff output for source %s with %d resources changed",
			data.Get("source").(string),
			len(diffs),
		)

		results = map[string]interface{}{}
		for _, diff := range diffs {
			results[diff.Name] = sanitizeDiff(diff.ClippedRawDiff(3000))
		}

		if len(results) > 0 {
			if err := data.SetNew(
				"diff",
				results,
			); err != nil {
				return err
			}
		} else {
			log.Infof(
				"Not updating diff field for source %s since diffs are empty",
				data.Get("source").(string),
			)
		}
	} else {
		log.Infof(
			"Resources have not changed for source %s",
			data.Get("source").(string),
		)
	}

	return nil
}

func resourceProfileUpdate(
	ctx context.Context,
	data *schema.ResourceData,
	provider interface{},
) diag.Diagnostics {
	log.Infof("Running update for source %s", data.Get("source").(string))
	diffValue := data.Get("diff").(map[string]interface{})

	// Null out diff so it's not persisted and we get a clean diff for the next apply.
	data.Set("diff", map[string]interface{}{})

	if len(diffValue) > 0 {
		providerCtx := provider.(*providerContext)
		expandResult, err := providerCtx.expand(ctx, data)
		if err != nil {
			return diag.FromErr(err)
		}
		defer providerCtx.cleanExpanded(expandResult)

		results, err := providerCtx.apply(ctx, expandResult.expandedDir)
		log.Infof(
			"Apply results for source %s (err=%+v): %s",
			data.Get("source").(string),
			err,
			string(results),
		)

		if err != nil {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  err.Error(),
					Detail:   string(results),
				},
			}
		}
	} else {
		log.Infof(
			"Diff is empty for source %s, so not running apply",
			data.Get("source").(string),
		)
	}

	return resourceProfileRead(ctx, data, provider)
}

func resourceProfileDelete(
	ctx context.Context,
	data *schema.ResourceData,
	provider interface{},
) diag.Diagnostics {
	log.Infof("Running no-op delete for source %s", data.Get("source").(string))

	// TODO: Support real deletes?
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "The kubeapply provider will not actually delete anything; please delete manually if needed",
			Detail:   fmt.Sprintf("Source: %s", data.Get("source").(string)),
		},
	}
}

func sanitizeDiff(rawDiff string) string {
	return sanitizationRegexp.ReplaceAllString(rawDiff, "${1}${2}: OMITTED")
}
