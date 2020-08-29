# Token Bucket implementation in Go

This is a pretty basic implementation of a token bucket for rate limiting in Go. When run, it'll start a webserver on port 8100
that has two endpoints:

localhost:8100/register_key will create a bucket and return back the UUID key for that bucket. You can click on that to
try to consume a token for that bucket.

localhost:8100/use_token?uuid=<uuid> tries to consume a token. It will let you know how many tokens you had left or 
return a 422 with a message if you have no tokens available and are rate limited.

It spits out a bunch of extra info on each page in an attempt to "show the work" as it were.