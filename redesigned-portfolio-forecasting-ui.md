# Preface

Portfolio Forecasting UI has been under heavy development through 2015, and this document summarises throughs and ideas that have been adopted, helping the new hires, especially in Forecasting UI team, to catch up easily.

# Engineering Introduction

## UI Foundation

### Connections & Shardings

UI initializes db connections through lib/connection_ext, manipulating connection pools, connection lifecycles and shardings.

Preseudo Code to initialize the MRM_AF shardings is as follows:

```
if use_connection :forecast?
  if has db configuration named 'forecast' in database.yml?
    if lu_forecast_network_shard_assignment has any record with data_source='MRM_AF'?
      build sharding according to 'lu_forecast_network_shard_assignment', 'lu_forecast_shard_vip_assignment'
    end
  end
end
```

i.e. If network sharding assignments exist in lu_forecast_network_shard_assignment

data_source|network_id|sharding_id
-----------|----------|-----------
MRM_AF|1|1

It indicates all forecasting related queries on network_id = 1, will go to sharding 1:

data_source|sharding_id|host|port|description
-----------|-----------|----|----|-----------
MRM_AF|1|h1|p1|d1

And sharding instances uses the same username/password pairs to the virtual db configuration in database.yml

Currently, UI maintains 4 types of shardings, which follows similar rules to MRM_AF shardings:

sharding keys|data sources|shardings|sharding assignments
-------------|------------|---------|--------------------
reporting|MRM_RPT|lu_shard_vip_assignment|lu_network_shard_assignment
forecast|MRM_AF|lu_forecast_shard_vip_assignment|lu_forecast_network_shard_assignment
fwrpm_partner_rpt|MRM_RPM_PARTNER|lu_shard_vip_assignment|lu_network_shard_assignment
fwrpm_adv_rpt|MRM_ADV|lu_shard_vip_assignment|lu_network_shard_assignment

## Schema Design

## Dataflow

## Contribute
