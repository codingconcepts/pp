.PHONY: test

test:
	go test ./... -v

test_script:
	echo "#!/bin/bash\n\necho "OK"" > test_script
	tar -zcvf ./test/test_v1.0.0_darwin_arm64.tar.gz ./test_script ;\

decompress_test_script:
	(cd test && tar -xvf test_v1.0.0_darwin_arm64.tar.gz)