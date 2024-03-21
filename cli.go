package main

import (
	"flag"
)

type CommandLine struct {
	Repo       string
	NumRecords int
	RepoOwner  string
	SearchTerm string
	LLM        string
}

func cliHandler() CommandLine {
	repoPtr := flag.String("repo", "aws-sdk-go-v2", "Repo name to search for releases.")
	numRecordsPtr := flag.Int("releases", 10, "Number of previous releases to search.")
	repoOwnerPtr := flag.String("owner", "aws", "Repo owner to search for releases.")
	searchTerm := flag.String("q", "costexplorer", "Search term to filter releases by.")

	flag.Parse()

	return CommandLine{
		Repo:       *repoPtr,
		NumRecords: *numRecordsPtr,
		RepoOwner:  *repoOwnerPtr,
		SearchTerm: *searchTerm,
	}

}
