# Pawndex

Pawn package list aggregator - [API live here](http://api.sampctl.com)

This small app simply uses the GitHub search API to find Pawn repositories (not SourcePawn, that language ID is too
polluted with unrelated repositories). It then places the results onto a queue which is slowly read from (once a second,
to avoid hammering the GitHub API too much). When the consumer reads from this queue, it checks if the repo contains a
[Package Definition File](https://github.com/Southclaws/sampctl/wiki/Package-Definition-Reference) and if it does,
passes that to the indexing queue to be listed on the API. If there is no package definition, it checks if there are any
.inc files in the top-level and if so, indexes the repo.

## Usage

Run your own instance with docker. Simply clone this repo, create an `.env` file containing:

```env
PAWNDEX_BIND=0.0.0.0:80
PAWNDEX_GITHUBTOKEN=abc123
PAWNDEX_SEARCHINTERVAL=1h
PAWNDEX_SCRAPEINTERVAL=30s
PAWNDEX_DATABASEPATH=pawndex.db
LOG_LEVEL=debug
```

- Bind is the interface to bind to, inside the container this is always 0.0.0.0:80.
- GitHub Token is your GitHub API token to circumvent rate limits
- Search Interval is the time between each query for GitHub Pawn repositories

Then run `make run` to run a production instance of Pawndex.
