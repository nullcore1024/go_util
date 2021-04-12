#ifndef _UTIL_H_
#define _UTIL_H_
extern int sum(int a,int b);
extern int greet(const char *name, int year, char *out); 
extern int rsaEncrypt(char* dest, int *destSize, char *source, int srcSize, const char* pszPulibcKeyN, const char* pszPulibcKeyE);
#endif
