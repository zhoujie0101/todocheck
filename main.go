package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/preslavmihaylov/todocheck/authmanager"
	"github.com/preslavmihaylov/todocheck/authmanager/authmiddleware"
	todocheckerrors "github.com/preslavmihaylov/todocheck/checker/errors"
	"github.com/preslavmihaylov/todocheck/config"
	"github.com/preslavmihaylov/todocheck/fetcher"
	"github.com/preslavmihaylov/todocheck/issuetracker"
	"github.com/preslavmihaylov/todocheck/traverser/todoerrs"
)

// TODO:
// * Add a --closes option which indicates that an issue is to be closed as a result of a PR
// * Add caching for task statuses
func main() {
	var basepath = flag.String("basepath", ".", "The path for the project to todocheck. Defaults to current directory")
	var cfgPath = flag.String("config", "", "The project configuration file to use. Will use the one from the basepath if not specified")
	var format = flag.String("format", "standard", "The output format to use. Available formats - standard, json")
	flag.Parse()

	localCfg, err := config.NewLocal(*cfgPath, *basepath)
	if err != nil {
		log.Fatalf("couldn't open configuration file: %s\n", err)
	}

	err = authmanager.AcquireToken(localCfg)
	if err != nil {
		log.Fatalf("couldn't acquire token from config: %s\n", err)
	}

	baseURL, err := issuetracker.BaseURLFor(localCfg.IssueTracker, localCfg.Origin)
	if err != nil {
		log.Fatalf("couldn't get base url for origin %s & issue tracker %s: %s\n",
			localCfg.Origin, localCfg.IssueTracker, err)
	}

	todoErrs := []*todocheckerrors.TODO{}
	f := fetcher.NewFetcher(baseURL, localCfg.IssueTracker, authmiddleware.For(localCfg))
	traverser := todoerrs.NewTraverser(f, localCfg.IgnoredPaths, func(todoErr *todocheckerrors.TODO) error {
		todoErrs = append(todoErrs, todoErr)
		return nil
	})

	err = traverser.TraversePath(*basepath)
	if err != nil {
		log.Fatalf("couldn't traverse basepath: %s", err)
	}

	if len(todoErrs) > 0 {
		printTodoErrs(todoErrs, *format)
		os.Exit(2)
	}
}

func printTodoErrs(errs []*todocheckerrors.TODO, format string) error {
	if len(errs) == 0 {
		if format == "json" {
			fmt.Println("[]")
		}

		return nil
	}

	if format == "standard" {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	} else if format == "json" {
		out := fmt.Sprintf("[%s", must(errs[0].ToJSON()))
		for _, err := range errs[1:] {
			out += fmt.Sprintf(",%s", must(err.ToJSON()))
		}

		out += "]"
		fmt.Println(out)
	}

	return errors.New("unrecognized output format: " + format)
}

func must(bs []byte, err error) []byte {
	if err != nil {
		panic(err)
	}

	return bs
}
