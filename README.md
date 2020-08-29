# Token Bucket rate limiting implementation in Go

Super simple token bucket implementation.   Hit /register_key to get a UUID, then pass that as a uuid= query paramter for use_token.   It should limit you to five requests in a burst,
and replenish tokens at a rate of one every five seconds.

localhost:8100/register_key will create a bucket and return back the UUID key for that bucket. You can click on that to
try to consume a token for that bucket.

localhost:8100/use_token?uuid=<uuid> tries to consume a token. It will let you know how many tokens you had left or 
return a 422 with a message if you have no tokens available and are rate limited. Hit refresh a bunch for
excitement!

It spits out a bunch of extra info on each page in an attempt to "show the work" as it were. It took me a bit to wrap my
head around the way this handled on all the cases as well as it does, so hopefully it'll
help you sort it out, too.
