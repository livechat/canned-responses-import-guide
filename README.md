# Import canned responses from CSV

Creates one Text canned response per CSV row, using the Configuration API v3.6. Create-only: it never updates or deletes.

See the [full setup guide](https://www.text.com/docs/guides/bulk-manage-canned-responses).

## Quickstart

Requires Go 1.27 or newer and a Text personal access token with the `canned_responses--all:rw` scope (to create responses in any group in your organization) or `canned_responses--groups:rw` (to create responses only in your groups).

Follow the setup guide above to create the token and prepare the CSV file.

```shell
git clone https://github.com/livechat/canned-responses-import-guide.git
cd canned-responses-import-guide
cp .env.example .env    # set TEXT_API_BASIC_AUTH
cp canned-responses.example.csv responses.csv
go run . --file responses.csv
```

A successful run prints `Created X of Y canned responses.`

The importer validates the whole CSV before it sends any request, so an invalid file makes no API calls. Row-level API errors are printed with their line numbers, and the command exits with a nonzero status if any response wasn't created.

**Every run creates every row again.** It doesn't check for duplicates or resume. After a partial import, check which responses exist in Text and remove those rows before retrying.

## Configuration

| Variable              | Required | Description                                                             |
| --------------------- | -------- | ----------------------------------------------------------------------- |
| `TEXT_API_BASIC_AUTH` | Yes      | `Basic ` followed by the Base64-encoded `account_id:PAT` value from Text. |

Set it in `.env`, or export it in your shell instead.

## CSV format

The header must be exactly `text,shortcuts,group_id,is_private`, in that order. The setup guide documents each column. `canned-responses.example.csv` is a working example.

The Configuration API names the shortcuts field `tags`. The importer maps the CSV `shortcuts` column to that field, to match what the Text app calls it.

## Tests

```shell
go test ./...
```

## License

This project is licensed under the [MIT License](LICENSE).

---

See the full [Text documentation reference](https://www.text.com/docs).
