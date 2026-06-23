# Monitor Service Module Status

## Scope

Linux host monitoring over SSH command snapshots.

## Important Files

- `service.go`

## Current State

Collects hostname, CPU load percentage, memory percentage, root disk percentage, top 5 processes by memory/CPU, network byte rates, and filesystem partitions.

## Known Work

Net speed is calculated between backend samples; the first sample normally reports zero rates.
