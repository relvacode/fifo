Direct streaming to and from remote object store providers like S3 for legacy commands

[![Build Status](https://travis-ci.org/relvacode/fifo.svg?branch=master)](https://travis-ci.org/relvacode/fifo)

### How It Works

Rewrite a file argument, stdin, stdout or stderr to a remote object URL such as S3.

It will create a named pipe in a temporary directory and will read or write data to the named pipe for a wrapped command to execute on.

It's an easy to use single binary distributable with zero dependencies. No need for FUSE or large temporary staging directories to store intermediate files before downloading or uploading to S3.

### Install

#### Mac OS

```
brew tap relvacode/fifo
brew install fifo
```

#### Linux

Replace `{version}` with your preffered [release](https://github.com/relvacode/fifo/releases)

```
curl -Ls https://github.com/relvacode/fifo/releases/download/v{version}/fifo_{version}_linux_x86_64.tar.gz | tar -xvzf - -C /usr/bin/ fifo
```

#### Docker

```
docker run relvacode/fifo
```

### Examples

__Backup a directory to a tar archive in S3 using the current date__

```
fifo -t archive=s3://bucket/archive-@{date}.tar.gz -- tar -C /directory -cvz . -f %{archive}
```

__Calculate the MD5 sum of a file in S3 using openssl__

```
fifo -s input=s3://bucket/file.txt -- openssl md5 %{input}
```

__Grep a file in S3 and upload the matches to S3__

```
fifo -s log=s3://bucket/log-file.txt --stdout s3://bucket/grepped.txt -- grep something %{log}
```

### Sources and Targets

Input and output targets are described as regular URLs. 

The scheme of the URL marks which source or target provider is used to find the object.

#### Templates

`@{ function }` templates can be used anywhere in a URL. 

These will get replaced with the return value of the function.

| Function | Returns |
| -------- | ------- |
| `date` | `YYYY-MM-dd` of the current date |
| `time` | `HH:mm:ss` of the current time |
| `datetime` | `YYYY-MM-ddTHH:mm:ss` of the current date-time |
| `hostname` | The hostname as reported by the OS |
| `uid` | UID of the current user |
| `gid` | GID of the current user |
| `random` | A six digit string from a random number between 000000 and 999999 (inclusive) |
#### Providers

##### `file://`

Opens or creates a file on the local filesystem

```
file://./log.txt
```

| Query Parameter | Behaviour |
| --------------- | --------- |
| `?append` | Append to the end of a file if it already exists. Defaults to truncate the file before writing |
| `?chmod` | When creating a file use these chmod style permissions. Defaults to `0644` |


##### `s3://` `s3+insecure://`

Downloads or uploads an object in S3.

Requires the environment parameters `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` and `AWS_REGION` are set.

```
s3://bucket/path/to/file.txt
```

```
s3://bucket/path/to/file.txt?acl=public-read&type=text/plain
```

| Query Parameter | Behaviour |
| --------------- | --------- |
| `?acl` | Set an [Amazon S3 canned ACL](https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl) on the created object |
| `?type` | Set the Content-Type of the created object |

##### `http://` `https://`

Stream an HTTP(s) URL

```
https://httpbin.org/stream/1
```

## Considerations

  - The application must read every source stream in its entirety. Seeking is not supported.

  - If an application does not open a file for reading or does not fully consume the stream then `fifo` may block forever (due to the nature of named pipes in UNIX).

  - Targets are destroyed automatically on failure unless `--preserve` is enabled.

