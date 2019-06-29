Fast remote storage streaming for legacy executables

### How It Works

You define some sources and/or targets as a mapping of some tag to a URL of the source/target object.

Then, using templates define a command to execute using those sources and destinations. 

Internally, fifo will replace those tags with a real path on your system to a named pipes whose other end reads or writes directly to the remote object.

### Examples

__Backup a directory to a tar archive in S3__

```
fifo -t archive=s3://bucket/archive.tar.gz -- tar -C /directory -cvz . -f %{archive}
```

__Calculate the MD5 sum of a file in S3 using openssl__

```
fifo -s input=s3://bucket/file.txt -- openssl md5 %{input}
```

__Grep a file in S3 and upload the matches to S3__

```
fifo -s log=s3://bucket/log-file.txt --stdout s3://bucket/grepped.txt -- grep something %{log}
```

### Providers

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

#### `http://`, `https://`

Stream a HTTP URL

```
https://httpbin.org/stream/1
```

## Considerations

  - The application must read every source stream in its entirety. Seeking is not supported.

  - If an application does not open a file for reading or does not fully consume the stream then `fifo` may block forever (due to the nature of named pipes in UNIX).

  - Targets are destroyed automatically on failure unless `--preserve` is enabled.

