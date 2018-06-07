DB Scheme Idea
--------------
```
|Field              |Comment                                    |
|-------------------|-------------------------------------------|
|ID                 |                                           |
|jenkins_job_url    |                                           |
|commit_id          |                                           |
|application_name   |hash of the docker image or tar.xz archive |
|build_start_ts     |                                           |
|build_duration_sec |                                           |
|total_src_hash     |combined hash of all source files          |

#### sources table
|Field      |Comment                                    |
|-----------|-------------------------------------------|
|ID         |                                           |
|build_id   |                                           |
|filepath   |relative to workspace root directory       |
|hash       |                                           |

#### artifacts table
|Field     |Comment                                    |
|----------|-------------------------------------------|
|ID        |                                           |
|build_id  |                                           |
|url       |                                           |
|hash      |hash of the docker image or tar.xz archive |
|size_kb   |                                           |
```

[modeline]: # ( vi:set tabstop=4 ft=markdown shiftwidth=4 tw=80 expandtab spell spl=en : )
