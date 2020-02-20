GRADLE=./gradlew

javaexec:
	@ \
	echo "[#] Building Java executor" ; \
	$(GRADLE) app:execman:distZip

test:
	@ \
	echo "[#] Running javaee test" ; \
	$(GRADLE) test

clean:
	@ \
	echo "[#] Cleaning javaee" ; \
	$(GRADLE) clean

.PHONY: test clean
