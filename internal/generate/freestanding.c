/* Freestanding libc supplied by wasm2go, without WASI or QuickJS std/os. */
#define STB_SPRINTF_IMPLEMENTATION
#include "stb_sprintf.h"
#include "libc.c"
#include "malloc_tlsf.c"
