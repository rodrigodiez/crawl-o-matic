# Crawl-o-matic
Hi Monzo!

This is my version of a concurrency based HTTP/HTML crawler.

I added some features that were not included in the original requirements. There is a funny story behind this! I was banned from 2 of the biggest sites in spain: http://www.meneame.net and https://www.forocoches.com while testing my crawler.

It was awesome to see how the crawler was only limited by the available network bandwidth but from the user point of view a crawler that gets banned is useless, and I am a very much user centric developer so I added:

- Rate limiting
- Max concurrent connections

I could have spent many hours more adding features or polishing some rough edges but it's probably better to discuss them with you. I hope you like it!

# How to run
go run main.go -url https://www.monzo.com -output sitemap.txt -interval 1µs -maxConcurrent 20
