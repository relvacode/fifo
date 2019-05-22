# FiFo

Real-time data streaming to S3 and other back-ends using the power of named pipes.

Allows legacy applications to easily stream data to file stores like S3 without the need for FUSE file systems.

## How It Works

You define some sources and targets as a mapping of some name to a URL of the source/target file.

Using Go templates define what sources and targets to use as command line arguments.
`fifo` will then render the templates with a real path to a named pipe on the file system.

## Examples

__Calculate the MD5 sum of a file in S3 using openssl__

```
fifo -s input:s3://bucket/file.txt -- openssl md5 {{.input}}
```

__Grep a file in S3 and upload the matches to S3__

```
fifo -s log:s3://bucket/log-file.txt --stdout s3://bucket/grepped.txt -- grep something {{.log}}
```

## Providers

#### `file://`

Opens or creates a file on the local filesystem

```
file://./log.txt
```

#### `s3://`, `s3+insecure://`

Downloads or uploads an object in S3.

Requires the environment parameters `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` and `AWS_ENDPOINT` are set.

```
s3://bucket/path/to/file.txt
```

## Considerations

  - The application must read every source stream in its entirety. Seeking is not supported.

  - If an application does not open a file for reading or does not fully consume the stream then `fifo` may block forever (due to the nature of named pipes in UNIX).

  - Targets are destroyed automatically on failure unless `--preserve` is enabled.

