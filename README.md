# Nginx rate limiter research

This is my research of Nginx's rate limiter and how it may be bypassed.

Run
```shell
go test
```

Result

<!-- RESULT:BEGIN -->
```
# TestSimple

		limit_req_zone $request_uri zone=my_zone:1m rate=30r/m;
		server {
			listen 80;
			location / {
				limit_req zone=my_zone;
				try_files $uri /index.html;
			}
		}
	

     URL             Start time             Duration 
✅   /0              57:23.603            1.779108ms 
✅   /1              57:23.604            1.602907ms 
❌   /0              57:23.604            1.766509ms 
❌   /1              57:23.603            1.856109ms 

❌   /0              57:24.607            1.216005ms 
❌   /1              57:24.606            1.371505ms 
❌   /0              57:24.606            2.279109ms 
❌   /1              57:24.606            1.781507ms 

✅   /0              57:25.609             671.103µs 
❌   /1              57:25.609            1.176404ms 
✅   /1              57:25.609            1.637006ms 
❌   /0              57:25.609            1.830707ms 

# TestSimpleBurst

		limit_req_zone $request_uri zone=my_zone:1m rate=12r/m;
		server {
			listen 80;
			location / {
				limit_req zone=my_zone burst=2;
				try_files $uri /index.html;
			}
		}
	

     URL             Start time             Duration 
✅   /1              57:27.634            3.557414ms 
❌   /0              57:27.635             2.45381ms 
✅   /0              57:27.635            2.743911ms 
❌   /1              57:27.635            3.010212ms 
❌   /0              57:27.635            3.638115ms 
❌   /1              57:27.635            3.871216ms 
✅   /0              57:27.635          5.007713523s *
✅   /1              57:27.635          5.007838422s *
✅   /0              57:27.635         10.007951804s *
✅   /1              57:27.635         10.007590302s *

# TestSimpleBurstNodelay

		limit_req_zone $request_uri zone=my_zone:1m rate=30r/m;
		server {
			listen 80;
			location / {
				limit_req zone=my_zone burst=2 nodelay;
				try_files $uri /index.html;
			}
		}
	

     URL             Start time             Duration 
✅   /1              57:40.615            2.720413ms 
✅   /1              57:40.616            2.624712ms 
✅   /1              57:40.615            2.715512ms 
✅   /0              57:40.615           18.733088ms 
✅   /0              57:40.615           18.849589ms 
❌   /0              57:40.615            19.26449ms 
❌   /1              57:40.615           19.486292ms 
❌   /1              57:40.616            19.11699ms 
✅   /0              57:40.615           19.636692ms 
❌   /0              57:40.615           19.540592ms 

❌   /0              57:41.636            1.150205ms 
❌   /1              57:41.636            1.104905ms 
❌   /0              57:41.636            1.723008ms 
❌   /1              57:41.636            1.761608ms 

# TestBypassRateLimiterSmallZoneSize

		limit_req_zone $huge$request_uri zone=my_zone:32k rate=1r/m;
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
✅   /any            57:44.635            2.497811ms 
✅   /some           57:44.635            2.377811ms 
❌   /any            57:44.635            2.627412ms 
❌   /some           57:44.635            2.519812ms 

✅   /1              57:44.638             521.403µs 
✅   /2              57:44.639             436.802µs 
✅   /3              57:44.639             451.602µs 
✅   /4              57:44.64              477.902µs 
✅   /5              57:44.64              388.502µs 
✅   /1              57:44.641             377.002µs 
✅   /2              57:44.641             361.002µs 
✅   /3              57:44.641             541.402µs 
✅   /4              57:44.642             387.902µs 
✅   /5              57:44.642             505.703µs 
✅   /1              57:44.643             388.502µs 
✅   /2              57:44.643             445.802µs 
✅   /3              57:44.644             413.302µs 
✅   /4              57:44.644             346.602µs 
✅   /5              57:44.645             415.902µs 

# TestBypassRateLimiterSmallZoneSizeBurst

		limit_req_zone $huge$request_uri zone=my_zone:32k rate=12r/m;
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
❌   /x              57:47.614            2.855013ms 
❌   /x              57:47.614            3.058015ms 
✅   /x              57:47.614          5.007857246s *
✅   /x              57:47.614         10.013183052s *
✅   /x              57:47.615            1.684508ms 
✅   /2              57:47.715             2.25671ms 
✅   /3              57:47.715            2.466112ms 
✅   /4              57:47.715            1.964009ms 
✅   /5              57:47.715            1.602208ms 
✅   /1              57:47.715            2.730513ms 
✅   /x              57:47.815            1.876309ms 
❌   /x              57:47.815            1.844608ms 
❌   /x              57:47.815             2.19281ms 
✅   /x              57:47.815          5.001689203s *
✅   /x              57:47.815         10.002447287s *

PASS
ok  	github.com/maratori/nginx_rate_limiter	44.582s

```
<!-- RESULT:END -->