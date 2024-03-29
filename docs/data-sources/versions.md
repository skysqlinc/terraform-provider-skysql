---
page_title: "skysql_versions Data Source - terraform-provider-skysql"
subcategory: ""
description: |-
  SkySQL server versions
---

# skysql_versions (Data Source)

SkySQL server versions

## Example Usage

```terraform
# List all SkySQL versions
data "skysql_versions" "default" {}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `topology` (String)

### Read-Only

- `versions` (Attributes List) (see [below for nested schema](#nestedatt--versions))

<a id="nestedatt--versions"></a>
### Nested Schema for `versions`

Read-Only:

- `display_name` (String) The display name of the version
- `id` (String) The ID of the version
- `is_major` (Boolean) Whether the version is a major version
- `name` (String) The name of the version
- `product` (String) The product that uses the version
- `release_date` (String) The release date of the version
- `release_notes_url` (String) The URL to the release notes of the version
- `topology` (String) The topology that uses the version
- `version` (String) The version display name

