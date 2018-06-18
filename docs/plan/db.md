## DB Scheme

### build table
```
|Field              |Comment                                    |
|-------------------|-------------------------------------------|
|ID                 |                                           |
|application_name   |                                           |
|start_timestamp    |                                           |
|build_duration_msec| or end timestamp?                         |
|total_src_hash     |combined hash of all source files          |
|-------------------|-------------------------------------------|
```

### artifact table
```
|Field               |Comment                                    |
|--------------------|-------------------------------------------|
|ID                  |                                           |
|name                |                                           |
|build_id            |                                           |
|type                |Docker or File                             |
|url                 |                                           |
|hash                |format: <type>:<sum>, not normalized       |
|size_kb             |                                           |
|upload_duration_msec|                                           |
---------------------|-------------------------------------------|
```

### artifact_src table
```
|Field      |Comment                                    |
|-----------|-------------------------------------------|
|artifact_id|                                           |
|source_id  |relative to workspace root directory       |
|-----------|-------------------------------------------|
```

### sources table
```
|Field      |Comment                                    |
|-----------|-------------------------------------------|
|ID         |                                           |
|rel_path   |relative to workspace root directory       |
|hash       |format: <type>:<sum>, not normalized       |
|-----------|-------------------------------------------|
```


[modeline]: # ( vi:set tabstop=4 ft=markdown shiftwidth=4 tw=80 expandtab spell spl=en : )
