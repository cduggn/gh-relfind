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
	repoPtr := flag.String("r", "", "Repo name to search for releases.")
	numRecordsPtr := flag.Int("n", 10, "Number of previous releases to search.")
	repoOwnerPtr := flag.String("o", "", "Repo owner to search for releases.")
	searchTerm := flag.String("k", "", "Search term to filter releases by.")

	flag.Parse()

	return CommandLine{
		Repo:       *repoPtr,
		NumRecords: *numRecordsPtr,
		RepoOwner:  *repoOwnerPtr,
		SearchTerm: *searchTerm,
	}

}
