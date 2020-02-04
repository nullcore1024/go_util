#include <stdio.h>
#include <stdint.h>
#include <string.h>
#ifndef LOG_DEBUG
#define LOG_DEBUG
#define log(fmt, ...) printf("%s(%d): ", __func__,__LINE__); printf(fmt, ##__VA_ARGS__)
#else
#define log(fmt, ...)
#endif

size_t hash(char* str, size_t i)
{
	size_t h = 0;
	size_t magic_exp = 1;
	if (str != NULL)
	{
		while (i-- != 0)
		{
			magic_exp *= 101;
			h += magic_exp + *str;
			++str;
		}
	}

	return h;
}

char* rabin_karp(char* s, char* find)
{
	char* p = NULL;

	if (s != NULL && find != NULL)
	{
		size_t n = strlen(s);
		size_t m = strlen(find);

		if (n > m)
		{
			size_t hfind = hash(find, m);

			char* end = s + (n - m + 1);
			for (char* i = s; i < end; ++i)
			{
				size_t hs = hash(i, m);
				if (hs == hfind)
				{
					if (strncmp(i, find, m) == 0)
					{
						p = i;
						break;
					}
				}
			}
		}
	}

	return p;
}

int main(int argc, char **argv)
{
	char *index = rabin_karp("cddeabcdedffeed", "cde");
	printf("%s\n", index);
	return 0;
}


