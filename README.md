# ratelimit

Super simple token bucket implementation.   Hit /register_key to get a UUID, then pass that as a uuid= query paramter for use_token.   It should limit you to five requests in a burst,
and replenish tokens at a rate of one every five seconds.
