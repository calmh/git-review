package main

import "strings"

type changes []change

type change struct {
	state string
	file  string
}

func (c *change) refresh() {
	line := mustCmd("git", "status", "--porcelain", "--untracked-files=all", c.file)
	c.state = line[0:2]
}

func allChanges() changes {
	files := mustCmd("git", "status", "--porcelain", "--untracked-files=all")
	tmp := strings.Split(files, "\n")
	changes := make(changes, 0, len(tmp))
	for _, line := range tmp {
		if line == "" {
			continue
		}
		changes = append(changes, change{
			state: line[0:2],
			file:  line[3:],
		})
	}
	return changes
}

func (c changes) done() int {
	done := 0
	for _, ch := range c {
		switch ch.state[0] {
		case 'M', 'A':
			done++
		}
	}
	return done
}
