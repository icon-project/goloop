/*
 */

#include "jni.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <signal.h>
#include <dirent.h>
#include <ctype.h>
#include <sys/types.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>

#include "foundation_icon_ee_ipc_Client.h"

#define RESTARTABLE(_cmd, _result) do { \
  do { \
    _result = _cmd; \
  } while((_result == -1) && (errno == EINTR)); \
} while(0)


int GetLastErrorString(char *buf, int len)
{
    if (errno == 0) return 0;

    const char *s = strerror(errno);
    size_t n = strlen(s);
    if (n >= len) {
        n = len - 1;
    }
    strncpy(buf, s, n);
    buf[n] = '\0';
    return n;
}

/*
 * Throw a Java exception by name.
 */
JNIEXPORT void JNICALL
JNU_ThrowByName(JNIEnv *env, const char *name, const char *msg)
{
    jclass cls = (*env)->FindClass(env, name);

    if (cls != 0) /* Otherwise an exception has already been thrown */
        (*env)->ThrowNew(env, cls, msg);
}

/* Throw an exception by name, using the string returned by
 * GetLastErrorString for the detail string.  If the last-error
 * string is NULL, use the given default detail string.
 */
JNIEXPORT void JNICALL
JNU_ThrowByNameWithLastError(JNIEnv *env, const char *name,
                             const char *defaultDetail)
{
    char buf[256];
    int n = GetLastErrorString(buf, sizeof(buf));
    if (n > 0) {
        JNU_ThrowByName(env, name, buf);
    } else {
        JNU_ThrowByName(env, name, defaultDetail);
    }
}

/*
 * Class:     foundation_icon_ee_ipc_Client
 * Method:    socket
 * Signature: ()I
 */
JNIEXPORT jint JNICALL Java_foundation_icon_ee_ipc_Client_socket
  (JNIEnv *env, jclass cls)
{
    int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd == -1) {
        JNU_ThrowByNameWithLastError(env, "java/io/IOException", "socket");
    }
    return (jint)fd;
}

/*
 * Class:     foundation_icon_ee_ipc_Client
 * Method:    connect
 * Signature: (ILjava/lang/String;)V
 */
JNIEXPORT void JNICALL Java_foundation_icon_ee_ipc_Client_connect
  (JNIEnv *env, jclass cls, jint fd, jstring path)
{
    jboolean isCopy;
    const char* p = (*env)->GetStringUTFChars(env, path, &isCopy);
    if (p != NULL) {
        struct sockaddr_un addr;
        socklen_t sockLen = sizeof(addr);
        int err = 0;

        addr.sun_family = AF_UNIX;
        strcpy(addr.sun_path, p);
        if (connect(fd, (struct sockaddr*)&addr, sockLen) == -1) {
            err = errno;
        }

        if (isCopy) {
            (*env)->ReleaseStringUTFChars(env, path, p);
        }

        /*
         * If the connect failed then we throw the appropriate exception
         * here (can't throw it before releasing the string as can't call
         * JNI with pending exception)
         */
        if (err != 0) {
            if (err == ENOENT) {
                JNU_ThrowByName(env, "java/io/FileNotFoundException", NULL);
            } else {
                char* msg = strdup(strerror(err));
                JNU_ThrowByName(env, "java/io/IOException", msg);
                if (msg != NULL) {
                    free(msg);
                }
            }
        }
    }
}

/*
 * Class:     foundation_icon_ee_ipc_Client
 * Method:    close
 * Signature: (I)V
 */
JNIEXPORT void JNICALL Java_foundation_icon_ee_ipc_Client_close
  (JNIEnv *env, jclass cls, jint fd)
{
    int res;
    shutdown(fd, SHUT_RDWR);
    RESTARTABLE(close(fd), res);
}

/*
 * Class:     foundation_icon_ee_ipc_Client
 * Method:    read
 * Signature: (I[BII)I
 */
JNIEXPORT jint JNICALL Java_foundation_icon_ee_ipc_Client_read
  (JNIEnv *env, jclass cls, jint fd, jbyteArray ba, jint off, jint baLen)
{
    unsigned char buf[128];
    size_t len = sizeof(buf);
    ssize_t n;

    size_t remaining = (size_t)(baLen - off);
    if (len > remaining) {
        len = remaining;
    }

    RESTARTABLE(read(fd, buf, len), n);
    if (n == -1) {
        JNU_ThrowByNameWithLastError(env, "java/io/IOException", "read");
    } else {
        if (n == 0) {
            n = -1;  // EOF
        } else {
            (*env)->SetByteArrayRegion(env, ba, off, (jint)n, (jbyte *)(buf));
        }
    }
    return n;
}

/*
 * Class:     foundation_icon_ee_ipc_Client
 * Method:    write
 * Signature: (I[BII)V
 */
JNIEXPORT void JNICALL Java_foundation_icon_ee_ipc_Client_write
  (JNIEnv *env, jclass cls, jint fd, jbyteArray ba, jint off, jint bufLen)
{
    size_t remaining = bufLen;
    do {
        unsigned char buf[128];
        size_t len = sizeof(buf);
        int n;

        if (len > remaining) {
            len = remaining;
        }
        (*env)->GetByteArrayRegion(env, ba, off, len, (jbyte *)buf);

        RESTARTABLE(write(fd, buf, len), n);
        if (n > 0) {
           off += n;
           remaining -= n;
        } else {
            JNU_ThrowByNameWithLastError(env, "java/io/IOException", "write");
            return;
        }
    } while (remaining > 0);
}
