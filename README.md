git-review
==========

This is a small tool to aid code reviews of pull requests. 

(Add problem statement here.)

Pricinciple of Operation
------------------------

Reviewing large and long running pull requests can be painful. git-review helps that process by:

  - Keeping track of the pull request in a local branch, by default called `pr/1234` for a pull request #1234.
  - Keeping track of what changes you have reviewed in a *review branch*, by default called `review/1234`.
  - Stepping through each change in the pull request and presenting options to add, skip and so on.

The end result is that the `review/1234` branch contains the code that you have
already reviewed, `pr/1234` contains whatever is the latest code on the pull
request. You then get to review the diff since your last reviewed state whenever
the pull request is updated, regardless if this is via pushing new commits,
ammending existing commits and force pushing, or rebasing. Just run `git review 1234` to get the latest changes and start reviewing them.

Installation
------------

From source:

    $ go get github.com/calmh/git-review

Or use the binary from the releases page and put it in your path.

Usage
-----

Assuming that there is a pull request number 1234 that needs review;

    $ git review 1234

This fetches the pull request into a branch, creates a review branch, resets the
review branch to the merge base of the pull request (i.e. where it was created
from), checks out the pull request, and resets the index. (Phew.) This basically
means you have the pull request in it's current state as an uncommitted change,
as if you'd been up hacking on it all night.

git-review then launches into the review loop. Here you get to handle each changed file:

    [1/16] [ M] lib/model/progressemitter.go - [Diff, Edit, Add, Patch, Skip, dOne, Quit]? 

Your options here are:

  - **D**iff (or just enter): Show the diff for the current file.
  - **E**dit: Open the file in your editor.
  - **A**dd: Add the file to the index, essentially marking it as reviewed. This also makes git-review move on to the next file.
  - **P**atch: Add to the index using interactive addding. Use this to mark parts of a file as OK or reviewed.
  - **S**kip: View the next changed file, without marking this one as reviewed.
  - D**o**ne: Commit the staged changes as a review commit on the review branch, then exit git-review.
  - **Q**uit: Exit git-review.

The intended workflow here is for you to work through the changes, making notes
of things to improve (in a separate document, on paper, or as comments on Github
as you please), adding files or chunks of files as they are reviewed. Once all
changes have been reviewed, or you grow tired, chose "Done" to commit your
review results to the review branch.

When the pull request gets updated in the future, you can just run `git review
1234` again to get the latest changes and start reviewing. However the diffs are
now compared to your latest review commit, so you only get to see what changed
since you last looked at the code. *Awesome!*

There are some further commands:

  - `git review updated` - check all ongoing reviews for updated pull requests.
  - `git review unreview 1234` - remove the pull request and review branches for this pull request.
  - `git review done` - create the review commit, if you have continued review outside of git-review for example.
  - `git review` (without a pull request number) - continue reviewing, assuming you are on a review branch.

License
-------

MIT
