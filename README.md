- Install go with `brew install go`
- Make sure this is in your `.zshrc` or similar: `export PATH="$PATH:$(go env GOPATH)/bin"`
- Install with `go install github.com/kvist-no/translazy@v1.0.0`.

NB: Expects to find `DEEPL_API_KEY` in environment. You can set this using `export DEEPL_API_KEY=...`

You can run it with:

```bash
translazy --sync -k my_cool_key 'A cool english base translation'
```

Run `translazy` without any arguments to see options
