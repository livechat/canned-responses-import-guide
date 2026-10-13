# Import canned responses from CSV

A small, create-only Go example for the Text Configuration API v3.6. Each CSV row is sent to `https://api.livechatinc.com/v3.6/configuration/action/create_canned_response` to create one **chat-side** canned response.

## Run

1. Install Go 1.27 or newer.
2. Create a Text personal access token with the `canned_responses_write` scope. In Text, copy the Base64-encoded `account_id:PAT` value.
3. Copy `.env.example` to `.env` and replace the placeholder with that value. Alternatively, set `TEXT_API_BASIC_AUTH` in your environment to `Basic <base64(account_id:PAT)>`.
4. Copy `canned-responses.example.csv` to `responses.csv`, then edit the rows.
5. Run `go run . --file responses.csv`.

The tool validates the entire CSV **before sending requests**, then prints `Created X of Y canned responses.` An invalid file makes no API calls. Row-level API errors are printed with line numbers; the command exits with a nonzero status if any response was not created.

## CSV format

The header must be exactly `text,tags,group_id,is_private`, in that order:

| Column | Format |
| --- | --- |
| `text` | Required, nonblank response text. Quote it if it contains commas or line breaks. The API sanitizes text and can reject content that becomes empty. |
| `tags` | Required: at least one nonblank tag. Separate multiple tags with `|`, for example `greeting|support`. |
| `group_id` | Non-negative integer for an existing Text group; General may be `0`. Invalid/nonexistent group IDs are reported by the API. |
| `is_private` | Required `true` or `false`. |

**Every run creates every row again.** It does not check for duplicates or resume. After a partial import, check which responses were created in Text and remove those rows from the CSV before retrying. If a request times out, it may have created a response even without a result; check Text before retrying.
