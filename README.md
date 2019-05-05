# quick_sms
A simple service to send an sms to a phone number via email.

Built on top of:
- go
- echo

## About
The most common use case for something like this is when you need to send an
an SMS notification to a relatively small set of numbers, sometimes its better to use a larger service (_like TWILLIO_) if you need to send to a large set of numbers and the provider for the numbers is not known. Also this approach doesn't really work for messaging back and forth.

---

## Setup
* rename the `.env-example` to `.env` and replace the variables with your own
* update the `knownNumbers.json` file with numbers you might expect to use.
* if using gmail, the setup for unsafe apps needs to be set see [this link](https://devanswers.co/allow-less-secure-apps-access-gmail-account/)

---

## Dev
* make any changes to code
* use the `./scripts/run.sh` to run locally

---

## Docker

* `docker-compose up`

## Test
* use postman or something to POST a body to `https://127.0.0.1/sms`

**Example Payload**
```json
{
    "number":"5550003333",
    "message":"I have a belly button.",
    "provider":"T-Mobile" (optional)
}
```
if no provider is set, then the known numbers will be checked to get that info.
