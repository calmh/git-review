package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	diffByDefault = getConfigDefaultBool("review.diffByDefault", true)
	diffOptions   = strings.Fields(getConfigDefault("review.diffOptions", "--patch --stat --histogram"))
	editor        = getConfigDefault("review.editor", os.Getenv("EDITOR"))
	pager         = getConfigDefault("review.pager", os.Getenv("PAGER"))
	pullPattern   = getConfigDefault("review.pullBranch", "pr/%s")
	remote        = getConfigDefault("review.remote", "origin")
	revPattern    = getConfigDefault("review.revBranch", "review/%s")
	verbose       = os.Getenv("RTDEBUG") != ""
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		review()
		return
	}

	switch flag.Arg(0) {
	case "review":
		review()

	case "done":
		done()

	case "updated":
		updated()

	case "unreview":
		unreview(flag.Arg(1))

	default:
		num := flag.Arg(0)
		revBranch := fmt.Sprintf(revPattern, num)
		pullBranch := fmt.Sprintf(pullPattern, num)
		mustCmd("git", "fetch", "-u", "-f", remote, "refs/pull/"+num+"/head:"+pullBranch)

		if _, err := cmd("git", "rev-parse", "--verify", revBranch); err != nil {
			// Review branch does not exist. Create it at the merge base.
			base := strings.TrimSpace(mustCmd("git", "merge-base", "master", pullBranch))
			mustCmd("git", "checkout", "-b", revBranch, base)
		} else {
			// Review branch exists. Continue from there.
			mustCmd("git", "checkout", revBranch)
		}
		mustCmd("git", "checkout", pullBranch, "--", ".")
		mustCmd("git", "reset")
		review()
	}
}

// review runs the interactive review loop to look through and handle changes
// since last review, or since the merge base if this is the first review
func review() {
	br := bufio.NewReader(os.Stdin)
	for {
		changes := allChanges()
		if len(changes) == 0 {
			fmt.Println("Nothing to review")
			return
		}

		if len(changes) == changes.done() {
			fmt.Printf("Nothing to review - [dOne, Quit]? ")
			resp, err := br.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			resp = strings.TrimSpace(strings.ToLower(resp))
			if resp == "" {
				continue
			}
			switch resp[0] {
			case 'o':
				done()
				return

			case 'q':
				return
			}
		}

		for i, change := range changes {
			first := true

		again:
			switch change.state {
			case "M ", "A ":
				// This file has been fully staged, so it's reviewed. Skip
				// it.
				continue

			default:
				if diffByDefault && first {
					// This is the first time we're passing by this file, and
					// we're supposed to show the diff by default. Do so and
					// mark it as done so we don't show the diff twice next
					// time around.
					first = false
					diff(change)
				}

				// Present the options and read a command. The default
				// command on just enter is "diff".
				fmt.Printf("[%d/%d] [%s] %s - [Diff, Edit, Add, Patch, Skip, dOne, Quit]? ", i+1, len(changes), change.state, change.file)
				resp, err := br.ReadString('\n')
				if err != nil {
					log.Fatal(err)
				}
				resp = strings.TrimSpace(strings.ToLower(resp))
				if resp == "" {
					resp = "d"
				}

				// Handle the command.
				switch resp[0] {
				case 'd':
					diff(change)
					goto again

				case 'e':
					interact(editor, change.file)
					goto again

				case 'a':
					mustCmd("git", "add", change.file)

				case 'p':
					interact("git", "add", "-p", change.file)
					change.refresh()
					goto again

				case 's':
					continue

				case 'o':
					done()
					return

				case 'q':
					return
				}
			}
		}
	}
}

// diff displays a change in diff format, unless it's a completely new file
// in which case we just show it with the pager.
func diff(c change) {
	if c.state == "??" {
		interact(pager, c.file)
	} else {
		args := []string{"diff"}
		args = append(args, diffOptions...)
		args = append(args, c.file)
		interact("git", args...)
	}
}

// done creates a commit with the currently staged files, with a description
// of how much has been reviewed
func done() {
	changes := allChanges()
	if len(changes) == 0 {
		return
	}
	mustCmd("git", "commit", "-m", fmt.Sprintf("reviewed %d/%d", changes.done(), len(changes)))
}

// cmd runs the specified command and returns the output as a string
func cmd(bin string, args ...string) (string, error) {
	if verbose {
		log.Println(bin, strings.Join(args, " "))
	}
	cmd := exec.Command(bin, args...)
	bs, err := cmd.CombinedOutput()
	out := string(bs)
	return out, err
}

// mustCmd runs the command and returns the output as a string. If the
// command returns an error the output is printed and we exit with an error.
func mustCmd(bin string, args ...string) string {
	res, err := cmd(bin, args...)
	if err != nil {
		log.Println(bin, strings.Join(args, " "))
		log.Println(res)
		log.Fatal(err)
	}
	return res
}

// interact runs a command connected to the user's stdin/stdout, such as an
// editor or a pager.
func interact(bin string, args ...string) error {
	log.Println(bin, strings.Join(args, " "))
	cmd := exec.Command(bin, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// getConfigDefault returns the value of the named git config, or the default
// if no value has been set.
func getConfigDefault(cfg, defVal string) string {
	val, err := cmd("git", "config", cfg)
	if err != nil {
		return defVal
	}
	return strings.TrimSpace(val)
}

// getConfigDefaultBool returns the value of the named git config as a
// boolean, or the default if no value has been set.
func getConfigDefaultBool(cfg string, defVal bool) bool {
	val, err := cmd("git", "config", "--bool", cfg)
	if err != nil {
		return defVal
	}
	return strings.TrimSpace(val) == "true"
}

func hasUpdated(pr string) bool {
	pullBranch := fmt.Sprintf(pullPattern, pr)
	oldsha := strings.TrimSpace(mustCmd("git", "rev-parse", "--verify", pullBranch))

	tmpBranch := pullBranch + "-tmp"
	mustCmd("git", "fetch", remote, "refs/pull/"+pr+"/head:"+tmpBranch)
	newSha := strings.TrimSpace(mustCmd("git", "rev-parse", "--verify", tmpBranch))
	mustCmd("git", "branch", "-D", tmpBranch)

	return newSha != oldsha
}

// updated prints the pull requests that have changed since last review
func updated() {
	rePattern := strings.Replace(pullPattern, "%s", `(\d+)`, 1)
	exp := regexp.MustCompile(rePattern)
	res := mustCmd("git", "branch")
	lines := strings.Split(res, "\n")
	changed := 0
	reviews := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		m := exp.FindStringSubmatch(line)
		if len(m) == 2 {
			reviews++
			if hasUpdated(m[1]) {
				changed++
				fmt.Println("PR", m[1], "has advanced")
			}
		}
	}
	fmt.Printf("%d of %d pull requests with open reviews have advanced\n", changed, reviews)
}

// unreview removes the remove and pull request branches for a given pull
// request
func unreview(pr string) {
	mustCmd("git", "branch", "-D", fmt.Sprintf(pullPattern, pr))
	mustCmd("git", "branch", "-D", fmt.Sprintf(revPattern, pr))
}
