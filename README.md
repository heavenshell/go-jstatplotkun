go-jstatplotkun
===============

Visualize JVM's JStat log.

##Usage
```
$ go build
$ ./go-jstatplotkun jstat --path=./path/to/jstat_gc.log --date="2015-03-30 22:00:00"
```

##Output png
- Eden.png
- GcTime.png
- Old.png
- GcCount.png
- Heap.png
- Perm.png
- Survivor1.png
- Survivor0.png
