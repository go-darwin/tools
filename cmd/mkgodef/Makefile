install:
	CGO_CFLAGS='${CGO_CFLAGS}' CGO_LDFLAGS='${CGO_LDFLAGS}' CGO_ENABLED=1 go install -v -x -trimpath -tags='osusergo,netgo' -ldflags='-s -w "-extldflags=-no-pie -Wl,-rpath /opt/llvm/3.9/lib"' .
