# gh-relfind

`gh-relfind` is a simple project started as an experiment to understand how to use Anthropic's Claude 3 Sonnet. The model is invoked through AWS Bedrock. It attempts to replicate the release history search capability of GitHub but from the command line. The initial version uses a keyword search [filter](https://github.com/samber/lo) to filter results. Response data from the Github `ListReleases` API is sent to AWS Bedrock where Claude 3 parses the ListReleases body field. It detects the release version, change and package information. The results are then written to stdout. 

> **Note** 
The initial version works best against repos that publish detailed release notes. There are many examples of projects that only publish tag information and no release notes (the golang/go repository is one such example). In these cases, the results will be empty.

## Pre-requisites
The AWS default credential chain is used to authenticate the request. Ensure that you have the necessary permissions to access Claude 3 Sonnet through AWS Bedrock.

## Installation

```bash
git clone https://github.com/cduggn/gh-relfind.git
```

## Usage

```bash
go run ./... -k <keyword> -o <owner> -n <num releases to search> -r <repo> 

# example usage against the offical Go repository
go run ./... -k costexplorer -o aws -n 20 -r aws-sdk-go-v2

```



