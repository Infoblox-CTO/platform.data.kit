# Engineering Members (engineering_members) Core Data Set

List of Infoblox engineering organization members.

## Source

- **Ref:** `https://github.com/Infoblox-CTO/saas.docs/blob/main/docs/_data/engineering_export.yaml`
- **Structure:** Top-level key `apps:`, where each child key is an app ID (e.g. `my-service`). Each entry is an object with fields including `namespace`, `owner`, and `productOwner`.

Each entry in the file represents a single engineering team member and is structured as a YAML list of objects.

### Retrieval Process

1. **Parse**: Load the YAML list. Each top-level list item is one member record.
2. **Derive `user_id`**: For each record, strip `@infoblox.com` from the `work_email` value to produce the `user_id`.
3. **Normalize optional fields**: The `director` and `department_evp` fields are not present on all records (see field table below). Treat absent fields as `null` / empty string as appropriate for your use case.

## Dataset Fields

| Field              | Type   | Required | Description                                                                    | Example                     |
| ------------------ | ------ | -------- | ------------------------------------------------------------------------------ | --------------------------- |
| `user_id`        | string | derived  | Username portion of the work email (work_email with `@infoblox.com` removed) | `jsmith`                  |
| `preferred_name` | string | yes      | Member's preferred display name                                                | `Jane Smith`              |
| `work_email`     | string | yes      | Infoblox work email address                                                    | `jsmith@infoblox.com`     |
| `business_title` | string | yes      | Official job title                                                             | `Staff Software Engineer` |
| `manager_name`   | string | yes      | Full name of the direct manager                                                | `Alex Johnson`            |
| `manager_email`  | string | yes      | Work email of the direct manager                                               | `ajohnson@infoblox.com`   |
| `director`       | string | no       | Name of the Director in the reporting chain                                    | `Rachel Lee`              |
| `department_vp`  | string | yes      | Name of the VP-level leader for the department                                 | `Naveen Singh`            |
| `department_evp` | string | no       | Name of the EVP-level leader for the department                                | `Padmini Kao`             |

### Field Notes

- **`user_id`** is a derived field — it is not stored in the YAML source. Compute it at read time by removing `@infoblox.com` from `work_email`.
- **`director`** is present in most records. Members who report directly to a VP or above may not have this field.
- **`department_evp`** is present in most records. Senior individual contributors or managers who report at the VP level may omit this field.
- **`manager_email`** can be used as a foreign key to look up the manager's own record within the same data set.

### Example Source Record

```yaml
- preferred_name: Jane Smith
  work_email: jsmith@infoblox.com
  business_title: Staff Software Engineer
  manager_name: Alex Johnson
  manager_email: ajohnson@infoblox.com
  director: Rachel Lee
  department_vp: Naveen Singh
  department_evp: Padmini Kao
```
