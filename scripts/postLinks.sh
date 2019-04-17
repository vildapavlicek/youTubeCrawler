#!/bin/bash
#used to post links to youTubeCrawler 
serverAddr=$1
n=$2

if [ $serverAddr == "" ]; then
	echo "First arg is server adress (localhost:8080), 2nd argument number of links to post (1-5)"
fi

case $2 in
1)
	curl -d '/watch?v=DT61L8hbbJ4' -X POST $serverAddr/api/v1/link
	;;
2)
	curl -d '/watch?v=DT61L8hbbJ4' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=Q3oItpVa9fs' -X POST $serverAddr/api/v1/link
	;;
3)
	curl -d '/watch?v=DT61L8hbbJ4' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=Q3oItpVa9fs' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=Yywb2E9t1sM' -X POST $serverAddr/api/v1/link
	;;
4)
	curl -d '/watch?v=DT61L8hbbJ4' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=Q3oItpVa9fs' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=Yywb2E9t1sM' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=obkWmcABqsM' -X POST $serverAddr/api/v1/link
	;;
5)
	curl -d '/watch?v=DT61L8hbbJ4' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=Q3oItpVa9fs' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=Yywb2E9t1sM' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=obkWmcABqsM' -X POST $serverAddr/api/v1/link
	sleep 1
	curl -d '/watch?v=UOxkGD8qRB4' -X POST $serverAddr/api/v1/link
	;;
*)
	echo "First arg is server adress (localhost:8080), 2nd argument number of links to post (1-5)"
esac
