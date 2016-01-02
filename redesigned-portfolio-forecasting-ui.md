# Preface

Portfolio Forecasting UI has been under heavy development through 2015, and this document summarises throughs and ideas that have been adopted, helping the new hires, especially in Forecasting UI team, to catch up easily.

# Engineering Introduction

## Dimensional Items

An Portfolio is associated to dimensional items up to 3 dimensions, and dimensional metrics depends on the traffic of dimension 1. And forecasting engine treats the dimension#2 and #3 as the same.

Dimension v.s. Traffic:

Dimension | Traffic
----------|--------
1 | dimension#1
2 | dimension#1 AND dimension#2
3 | dimension#1 AND dimension#2 AND dimension#3

Dimensional Items v.s. OLTP Tables

- items on dimension 1 will be persisted to both targeting criteria tree, including targeting_criteria, targeting_criteria_assignment and targeting_criteria_item_assignment, and forecast_portfolio_dimension_item_assignment

- items on dimension 2, 3 will be persisted to forecast_portfolio_dimension_item_assignment

i.e. Video Group broken down by Country further broken down by DMA may have following targeting items

Dimension | Items
----------|-------
Video Group| VG1, VG2
Country | US, UK
DMA | NY, LA

Therefore, traffic on *VG1 broken down US* may be larger than that on *sum of VG1 broken down by US further broken down by targeted items of DMA*.

![a screenshot of chart for the dimensional metrics]

There are a few special dimension types, and they are

- Multi-Targeted: there is only 1 record in forecast_portfolio_dimension_item_assignment, and it must be {dimension_type_id: 128, dimension_order: 1, dimension_value: -1}

- All Direct Children and RBP: sub_dimension_type_id is not null, i.e. {dimension_type_id: 6, dimension_order: 2, sub_dimension_type_id: 10, dimension_value: 1}

UI restrain clients to create invalid dimensional combinations through a decision table:

```
# modules/forecasting_mgmt/app/models/forecasting_mgmt/portfolio_break_down_restriction.rb#54

ForecastingMgmt::PortfolioBreakDownRestriction::DIMENSION_RESTRICTION
```

## Schema

OTLP

Table | Description
------|------------
forecast_portfolio | Subject of a portfolio
lu_portfolio_forecast_dimension_type | relationship of dimension type and criteria type
forecast_portfolio_dimension_item_assignment | dimensional item/type assignments, .i.e Site1 on dimension 1
label | enable clients to filter their portfolio by labels
label_assignment | label v.s. portfolio with assign_type = 'PORTFOLIO'
targeting_criteria | criterias
targeting_criteria_assignment | parent-child of targeting_criterias
targeting_criteria_item_assignment | item assignments for targeting


AF-IB




## Dataflow

## Contribute
