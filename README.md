# Crawl-o-matic
[![build](	https://img.shields.io/travis/rodrigodiez/crawl-o-matic/master.svg)](https://travis-ci.org/rodrigodiez/crawl-o-matic)
[![Go Report Card](https://goreportcard.com/badge/github.com/rodrigodiez/crawl-o-matic)](https://goreportcard.com/report/github.com/rodrigodiez/crawl-o-matic)
[![MIT License](https://img.shields.io/github/license/rodrigodiez/crawl-o-matic.svg)](https://github.com/rodrigodiez/crawl-o-matic/blob/master/LICENSE.md)

Hi Monzo!

This is my version of a concurrency based HTTP/HTML crawler.

I added some features that were not included in the original requirements. There is a funny story behind this! I was banned from 2 of the biggest sites in spain: http://www.meneame.net and https://www.forocoches.com while testing my crawler.

It was awesome to see how the crawler was only limited by the available network bandwidth but from the user point of view a crawler that gets banned is useless, and I am a very much user centric developer so I added:

- Rate limiting
- Max concurrent connections

I could have spent many hours more adding features or polishing some rough edges but it's probably better to discuss them with you. I hope you like it!

# How to run
Make sure you have `dep` available in your sistem and project dependencies are installed:

```
go get -u github.com/golang/dep/cmd/dep
dep ensure
```

Crawl 'Em All!

```
go run main.go -url https://www.monzo.com -output sitemap.txt -interval 1µs -maxConcurrent 20
```