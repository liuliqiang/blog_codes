#include <cstdlib>
#include <cstdio>
#include <cerrno>
#include <sys/mman.h>
#include <fcntl.h>
#include <unistd.h>  // 添加这个头文件

int main(void)
{
    int fd = open("hello", O_RDWR);
    if(fd < 0)
    {
        perror("open hello");
        exit(1);
    }
    void *p = mmap(NULL,6, PROT_WRITE, MAP_SHARED, fd, 0);
    if(p == MAP_FAILED)
    {
       perror("mmap"); // 程序进里面了，证明 mmap 失败
       exit(1);
    }
    printf("%p\n", p);
    close(fd);
    ((int*)p)[0] = 0x30313233;
    munmap(p, 6);
    return 0;

}