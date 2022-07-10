# Nginx rate limiter research

This is my research of Nginx's rate limiter and how it may be bypassed.

Run
```shell
go test
```

Result

<!-- RESULT:BEGIN -->
```
## TestSimple
    Just an example of rate limiter, no surprise.
    Each uri has it's own leaky bucket (without buffer) which drains each 2 seconds.

        limit_req_zone $uri zone=my_zone:1m rate=30r/m;
        server {
        	listen 80;
        	location / {
        		limit_req zone=my_zone;
        		try_files $uri /index.html;
        	}
        }

     URL             Start time             Duration 
✅   /0              36:40.464            1.278892ms 
✅   /1              36:40.464            1.319791ms 
❌   /0              36:40.464             1.42429ms 
❌   /1              36:40.464            1.383391ms 

❌   /1              36:41.466             480.196µs 
❌   /0              36:41.466             614.296µs 
❌   /0              36:41.466             775.094µs 
❌   /1              36:41.466             742.095µs 

❌   /0              36:42.467             441.197µs 
✅   /0              36:42.467             971.194µs 
✅   /1              36:42.467             529.096µs 
❌   /1              36:42.467             902.094µs 

## TestSimpleBurst
    Just an example of burst config, no surprise.
    It creates buffer for each leaky bucket.

        limit_req_zone $uri zone=my_zone:1m rate=12r/m;
        server {
        	listen 80;
        	location / {
        		limit_req zone=my_zone burst=2;
        		try_files $uri /index.html;
        	}
        }

     URL             Start time             Duration 
✅   /1              36:44.413             1.41979ms 
✅   /0              36:44.413            1.676989ms 
❌   /0              36:44.413            1.844988ms 
❌   /0              36:44.413            2.046286ms 
❌   /1              36:44.414            1.680988ms 
❌   /1              36:44.413           14.399001ms 
✅   /0              36:44.413          5.004899501s *
✅   /1              36:44.413          5.004509704s *
✅   /0              36:44.413         10.006220029s *
✅   /1              36:44.414         10.005958531s *

## TestSimpleBurstNodelay
    Just an example of nodelay config, no surprise.
    The same as previous, but all requests are processed at the same time.

        limit_req_zone $uri zone=my_zone:1m rate=30r/m;
        server {
        	listen 80;
        	location / {
        		limit_req zone=my_zone burst=2 nodelay;
        		try_files $uri /index.html;
        	}
        }

     URL             Start time             Duration 
✅   /0              36:56.437           14.031202ms 
✅   /0              36:56.437           13.958203ms 
✅   /0              36:56.437             14.3259ms 
✅   /1              36:56.437           14.148101ms 
✅   /1              36:56.437             14.2681ms 
✅   /1              36:56.437             14.2285ms 
❌   /0              36:56.437           14.180201ms 
❌   /0              36:56.437           14.402599ms 
❌   /1              36:56.437           14.180101ms 
❌   /1              36:56.437             14.2523ms 

❌   /0              36:57.452             810.295µs 
❌   /0              36:57.452             924.994µs 
❌   /1              36:57.452             731.495µs 
❌   /1              36:57.452             876.994µs 

## TestBypassRateLimiterSmallZoneSize
    It's possible to bypass rate limiter, if zone size is small.
    Nginx forgets about least recently used key, if zone is exhausted.
    It means hacker can cycle requests.

        limit_req_zone $huge$uri zone=my_zone:32k rate=1r/m;
        server {
        	listen 80;
        	location / {
        		set $x 1234567890;
        		set $y $x$x$x$x$x$x$x$x$x$x;
        		set $z $y$y$y$y$y$y$y$y$y$y;
        		set $huge $z$z$z$z$z;
        		limit_req zone=my_zone;
        		try_files $uri /index.html;
        	}
        }

     URL             Start time             Duration 
✅   /any            36:59.438             911.493µs 
✅   /some           36:59.438             1.34109ms 
❌   /any            36:59.438             1.40129ms 
❌   /some           36:59.438            1.302691ms 

✅   /1              36:59.44              253.198µs 
✅   /2              36:59.44              231.598µs 
✅   /3              36:59.44              306.898µs 
✅   /4              36:59.44              237.598µs 
✅   /5              36:59.441             247.598µs 
✅   /1              36:59.441             261.898µs 
✅   /2              36:59.441             233.498µs 
✅   /3              36:59.442             236.598µs 
✅   /4              36:59.442             207.399µs 
✅   /5              36:59.442             231.798µs 
✅   /1              36:59.442             222.999µs 
✅   /2              36:59.442             248.599µs 
✅   /3              36:59.443             250.899µs 
✅   /4              36:59.443             277.698µs 
✅   /5              36:59.443             229.399µs 

## TestBypassRateLimiterSmallZoneSizeBurst
    Here is what happens with requests waiting in burst's buffer, when zone is exhausted.
    I thought they should be rejected, but in fact they are processed.

        limit_req_zone $huge$uri zone=my_zone:32k rate=12r/m;
        server {
        	listen 80;
        	location / {
        		set $x 1234567890;
        		set $y $x$x$x$x$x$x$x$x$x$x;
        		set $z $y$y$y$y$y$y$y$y$y$y;
        		set $huge $z$z$z$z$z;
        		limit_req zone=my_zone burst=2;
        		try_files $uri /index.html;
        	}
        }

     URL             Start time             Duration 
✅   /x              37:01.425            1.082692ms 
❌   /x              37:01.425             1.38199ms 
❌   /x              37:01.425             1.53949ms 
✅   /x              37:01.425          5.006127802s *
✅   /x              37:01.425         10.002559449s *
✅   /1              37:01.526             818.194µs 
✅   /2              37:01.526            1.311291ms 
✅   /3              37:01.526            1.352591ms 
✅   /4              37:01.526            1.295391ms 
✅   /5              37:01.526            1.013293ms 
✅   /x              37:01.627            1.213292ms 
❌   /x              37:01.627            1.073692ms 
❌   /x              37:01.627            1.089193ms 
✅   /x              37:01.627          5.002115358s *
✅   /x              37:01.627         10.001785681s *

PASS
ok  	github.com/maratori/nginx_rate_limiter	40.496s

```
<!-- RESULT:END -->