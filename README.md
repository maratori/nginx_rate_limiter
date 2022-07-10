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
❌   /1              07:41.889             1.27321ms 
✅   /0              07:41.889            1.963715ms 
✅   /1              07:41.889            1.505112ms 
❌   /0              07:41.889            1.538312ms 

❌   /0              07:42.891             945.207µs 
❌   /0              07:42.891             938.507µs 
❌   /1              07:42.891             857.906µs 
❌   /1              07:42.891             1.26841ms 

❌   /0              07:43.893             1.31971ms 
✅   /0              07:43.893            1.733613ms 
✅   /1              07:43.893            1.781413ms 
❌   /1              07:43.893            1.609412ms 

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
✅   /1              07:46.554            2.456419ms 
❌   /1              07:46.554            2.473219ms 
✅   /0              07:46.553            2.592219ms 
❌   /1              07:46.553            2.789922ms 
✅   /1              07:46.554          5.007125625s *
✅   /0              07:46.553         10.007217199s *
✅   /1              07:46.553         10.007037497s *
❌   /0              07:46.556             716.106µs 
❌   /0              07:46.556             839.807µs 
✅   /0              07:46.556          5.004480204s *

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
✅   /1              07:58.529            1.333311ms 
✅   /1              07:58.529            1.012108ms 
✅   /0              07:58.528            2.979423ms 
✅   /0              07:58.528            3.179124ms 
✅   /1              07:58.527            3.460726ms 
❌   /0              07:58.528            2.731921ms 
✅   /0              07:58.528           20.329056ms 
❌   /0              07:58.529           20.096954ms 
❌   /1              07:58.529           19.536049ms 
❌   /1              07:58.529           19.936253ms 

❌   /0              07:59.549             569.205µs 
❌   /0              07:59.549            1.171609ms 
❌   /1              07:59.549            1.104808ms 
❌   /1              07:59.549             952.907µs 

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
✅   /any            08:01.519            1.924615ms 
✅   /some           08:01.519            1.592012ms 
❌   /any            08:01.519            2.005315ms 
❌   /some           08:01.519            1.720013ms 

✅   /1              08:01.521             397.403µs 
✅   /2              08:01.522             446.204µs 
✅   /3              08:01.522             324.103µs 
✅   /4              08:01.523             394.303µs 
✅   /5              08:01.523             374.803µs 
✅   /1              08:01.524             327.603µs 
✅   /2              08:01.524             391.703µs 
✅   /3              08:01.524             387.403µs 
✅   /4              08:01.525             342.003µs 
✅   /5              08:01.525             413.903µs 
✅   /1              08:01.526             375.802µs 
✅   /2              08:01.526             334.702µs 
✅   /3              08:01.526             375.803µs 
✅   /4              08:01.527             368.903µs 
✅   /5              08:01.527             441.903µs 

## TestBypassRateLimiterSmallZoneSizeBurst
    It's an example of what happens with requests waiting in burst's buffer, when zone is exhausted.
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
✅   /x              08:03.531            1.606312ms 
❌   /x              08:03.531            1.875514ms 
❌   /x              08:03.531            2.011516ms 
✅   /x              08:03.53           5.006337333s *
✅   /x              08:03.53          10.007331134s *
✅   /5              08:03.63              745.706µs 
✅   /1              08:03.63             2.176917ms 
✅   /3              08:03.631            2.097816ms 
✅   /4              08:03.631            1.964015ms 
✅   /2              08:03.63             3.611627ms 
✅   /x              08:03.731            1.746313ms 
❌   /x              08:03.731            1.825814ms 
❌   /x              08:03.732            2.135216ms 
✅   /x              08:03.732          5.001752899s *
✅   /x              08:03.732         10.001784194s *

PASS
ok  	github.com/maratori/nginx_rate_limiter	42.801s

```
<!-- RESULT:END -->