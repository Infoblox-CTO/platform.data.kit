{{ config(
    materialized='table'
) }}

-- Summarise the example_table seeded by dk dev up.
-- Produces one row per name with the earliest creation timestamp.

select
    name,
    min(created_at) as first_seen,
    count(*)        as row_count
from {{ source('public', 'example_table') }}
group by name
order by first_seen
