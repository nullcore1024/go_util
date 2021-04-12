#include <openssl/rsa.h>
#include <openssl/bn.h>
#include <openssl/err.h>
#include <stdint.h>
#include <string.h>
#include "util.h"

int sum(int a,int b){
    return (a+b);
}

int greet(const char *name, int year, char *out) {
    int n;

    n = sprintf(out, "Greetings, %s from %d! We come in peace :)", name, year);

    return n;
}

#define __CLASS__ "rsa"
#define log(fmt, ...) fprintf(fp, "%s(%d): ", __func__,__LINE__); fprintf(fp, fmt, ##__VA_ARGS__); fprintf(fp, "\n")
#define log_error(fmt, ...) fprintf(fp, "%s(%d): ", __func__,__LINE__); fprintf(fp, fmt, ##__VA_ARGS__);fprintf(fp, "\n")

int rsaEncrypt(char* dest, int *destSize, char *source, int srcSize, const char* pszPulibcKeyN, const char* pszPulibcKeyE)
{
    FILE *fp = NULL;
    fp = fopen("test.log", "w+");

    fprintf(fp, "hello\n");
    log("src size:%d\n", srcSize);
    log("n size:%ld", strlen(pszPulibcKeyN));
    log("e size:%ld", strlen(pszPulibcKeyE));
    int i = 0;
    log("dump src=====:%ld\n", strlen(pszPulibcKeyE));
    for (; i < srcSize; i ++) {
        fprintf(fp, "%d", source[i]);
    }
    int ret = -1;
    RSA* rsa = RSA_new();
    do {
        BIGNUM* n= NULL, *e=NULL;
        BN_hex2bn(&n, pszPulibcKeyN);
        BN_hex2bn(&e, pszPulibcKeyE);
        if(!n || !e) {
            log_error("[%s::%s] BN_bin2bn failed", __CLASS__, __FUNCTION__);
            ret = -2;
            break;
        }
        rsa->n = n;
        rsa->e = e;
        /*
        if(RSA_set0_key(rsa, n, e, nullptr) != 1) {
            log_error("[%s::%s] RSA_set0_key failed", __CLASS__, __FUNCTION__);
            break;
        }
        */

        int modulus_size = RSA_size(rsa);
        if(srcSize <= modulus_size - 11) {
            log("modules_size:%d", modulus_size);
            *destSize = modulus_size;
            if(RSA_public_encrypt(srcSize, source, dest, rsa, RSA_PKCS1_PADDING) != modulus_size)
            {
                log_error("[%s::%s] RSA_public_encrypt failed, return size not %d", __CLASS__, __FUNCTION__, modulus_size);
                ret = -3;
                break;
            }
        } else {
            log("2 modules_size:%d", modulus_size);
            char * part = (char *)malloc(1024);
            uint32_t offset = 0;
            while(offset < srcSize) {
                int flen = 0;
                if (modulus_size - 11 < (srcSize - offset)) {
                    flen = modulus_size - 11;
                } else {
                    flen = srcSize - offset;
                }
                if(RSA_public_encrypt(flen, source + offset, part, rsa, RSA_PKCS1_PADDING) != modulus_size)
                {
                    log_error("[%s::%s] RSA_public_encrypt failed, part, return size not %d", __CLASS__, __FUNCTION__, modulus_size);
                    ret = -4;
                    break;
                }
                memcpy(dest+offset, part, flen);
                offset += flen;
                memset(part, 0, flen);
            }
            if(offset != srcSize) {
                log_error("[%s::%s] failed, ", __CLASS__, __FUNCTION__);
                ret = -5;
                break;
            }
        }
        ret = 0;
    } while(0);

    if(rsa) {
        RSA_free(rsa);
    }

    return ret;
}
