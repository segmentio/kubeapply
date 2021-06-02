package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	log "github.com/sirupsen/logrus"
)

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
			"path": {
				Type:        schema.TypeString,
				Description: "Path to profile manifest files",
				Required:    true,
			},
			"parameters": {
				Type:        schema.TypeMap,
				Description: "Arbitrary parameters that will be used for profile expansion",
				Optional:    true,
			},
			"force_diff": {
				Type:        schema.TypeBool,
				Description: "Force a full diff against all resources even if no inputs changed",
				Optional:    true,
			},
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
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	log.Infof("Running create")
	providerCtx := m.(*providerContext)
	var diags diag.Diagnostics

	expandResult, err := providerCtx.expand(
		ctx,
		d.Get("path").(string),
		d.Get("parameters").(map[string]interface{}),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(
		"resources", expandResult.resources,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	// Just make up an id from the timestamp
	d.SetId(fmt.Sprintf("%d", time.Now().UnixNano()))

	log.Info("Create successful")
	return diags
}

func resourceProfileRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	log.Infof("Running read")
	var diags diag.Diagnostics

	// There's nothing to do here since we only read the remote kube state if we're doing a
	// diff.
	return diags
}

func resourceProfileCustomDiff(
	ctx context.Context,
	d *schema.ResourceDiff,
	m interface{},
) error {
	log.Infof("Running custom diff")
	providerCtx := m.(*providerContext)
	expandResult, err := providerCtx.expand(
		ctx,
		d.Get("path").(string),
		d.Get("parameters").(map[string]interface{}),
	)
	if err != nil {
		return err
	}

	log.Infof(
		"Found %d manifests with overall hash of %s",
		len(expandResult.manifests),
		expandResult.totalHash,
	)

	// Set resources
	if err := d.SetNew(
		"resources",
		expandResult.resources,
	); err != nil {
		return err
	}

	if d.HasChange("resources") || d.Get("force_diff").(bool) {
		log.Info("Resources have changed")

		results := cache.get(expandResult.totalHash)
		if results == nil {
			log.Info("No cache hit, recomputing diffs")

			if err := providerCtx.createNamespaces(ctx, expandResult.manifests); err != nil {
				return err
			}

			diffs, err := providerCtx.diff(ctx, expandResult.expandedDir)
			if err != nil {
				return err
			}
			log.Infof("Got structured diff output: %+v", len(diffs))

			results = map[string]interface{}{}
			for _, diff := range diffs {
				results[diff.Name] = diff.ClippedRawDiff(3000)
			}

			cache.set(expandResult.totalHash, results)
		} else {
			log.Info("Cache hit, not recomputing diffs")
		}

		if len(results) > 0 {
			if err := d.SetNew(
				"diff",
				results,
			); err != nil {
				return err
			}
		} else {
			log.Info("Not updating diff field since diffs are empty")
		}
	} else {
		log.Info("Resources have not changed")
	}

	return nil
}

func resourceProfileUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	log.Infof("Running update")

	// Null out diff so it's not persisted and we get a clean diff for the next apply.
	d.Set("diff", map[string]interface{}{})

	providerCtx := m.(*providerContext)
	expandResult, err := providerCtx.expand(
		ctx,
		d.Get("path").(string),
		d.Get("parameters").(map[string]interface{}),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	results, err := providerCtx.apply(ctx, expandResult.expandedDir)
	log.Infof("Apply results (err=%+v): %s", err, string(results))
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  err.Error(),
				Detail:   string(results),
			},
		}
	}

	return resourceProfileRead(ctx, d, m)
}

func resourceProfileDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	log.Infof("Running delete")

	// TODO: Support real deletes?
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "The kubeapply provider will not actually delete anthing; please delete manually if needed",
		},
	}
}
