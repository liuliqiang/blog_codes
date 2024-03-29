//
// Created by liuliqiang on 9/8/22.
//

#include <fuse3/fuse.h>
#include <cstdio>
#include <cstring>
#include <cerrno>
#include <fcntl.h>
#include <cstddef>
#include <cassert>

using namespace std;

#include <glog/logging.h>

static struct options {
    const char *filename;
    const char *contents;
    int show_help;
} options;

static void *hello_init(struct fuse_conn_info *conn,
                        struct fuse_config *cfg) {
    LOG(INFO) << "init invoked";
    cfg->kernel_cache = 1;
    return NULL;
}

class HelloClass {
public:
    static int hello_getattr(const char *path, struct stat *stbuf, struct fuse_file_info *fi);

    static int hello_readdir(const char *, void *, fuse_fill_dir_t, off_t, struct fuse_file_info *,
                             enum fuse_readdir_flags);
};

int HelloClass::hello_getattr(const char *path, struct stat *stbuf, struct fuse_file_info *fi) {
    LOG(INFO) << "getattr invoked";
    int ret = 0;

    printf("path is %s\n", path);
    memset(stbuf, 0, sizeof(struct stat));
    if (strcmp(path, "/") == 0) {
        stbuf->st_mode = S_IFDIR | 0755;
        stbuf->st_nlink = 2;
    } else if (strcmp(path + 1, options.filename) == 0) {
        stbuf->st_mode = S_IFREG | 0444;
        stbuf->st_nlink = 1;
        stbuf->st_size = strlen(options.contents);
    } else {
        ret = -ENOENT;
    }

    return ret;
}


int HelloClass::hello_readdir(const char *path,
                              void *buf,
                              fuse_fill_dir_t filler,
                              off_t offset,
                              struct fuse_file_info *fi,
                              enum fuse_readdir_flags flags) {
    LOG(INFO) << "readdir invoked";
    if (strcmp(path, "/") == 0) {
        return -ENOENT;
    }

    filler(buf, ".", NULL, 0, static_cast<fuse_fill_dir_flags>(0));
    filler(buf, "..", NULL, 0, static_cast<fuse_fill_dir_flags>(0));
    filler(buf, options.filename, NULL, 0, static_cast<fuse_fill_dir_flags>(0));

    return 0;
}


static int hello_open(const char *path,
                      struct fuse_file_info *fi) {
    LOG(INFO) << "open invoked";
    if (strcmp(path + 1, options.filename) != 0) {
        return -ENOENT;
    }

    if ((fi->flags & O_ACCMODE) != O_RDONLY) {
        return -EACCES;
    }

    return 0;
}

static int hello_read(const char *path,
                      char *buf,
                      size_t size,
                      off_t offset,
                      struct fuse_file_info *fi) {
    LOG(INFO) << "read invoked";
    if (strcmp(path + 1, options.filename) != 0) {
        return -ENOENT;
    }

    size_t len = strlen(options.contents);
    if (size_t(offset) < len) {
        if (offset + size > len) {
            size = len - offset;
        }
        memcpy(buf, options.contents + offset, size);
    } else {
        size = 0;
    }

    return size;
}


HelloClass hc;

static const struct fuse_operations hello_oper = {
        .getattr = hc.hello_getattr,
        .open = hello_open,
        .read = hello_read,
        .readdir = hc.hello_readdir,
        .init = hello_init,
};

#define OPTION(t, p)                           \
    { t, offsetof(struct options, p), 1 }
static const struct fuse_opt option_spec[] = {
        OPTION("--name=%s", filename),
        OPTION("--contents=%s", contents),
        OPTION("-h", show_help),
        OPTION("--help", show_help),
        FUSE_OPT_END
};

static void show_help(const char *progname) {
    printf("usage: %s [options] <mountpoint>\n\n", progname);
    printf("File-system specific options:\n"
           "    --name=<s>          Name of the \"hello\" file\n"
           "                        (default: \"hello\")\n"
           "    --contents=<s>      Contents \"hello\" file\n"
           "                        (default \"Hello, World!\\n\")\n"
           "\n");
}

int main(int argc, char *argv[]) {
    google::InitGoogleLogging(argv[0]);
    LOG(INFO) << "Start my fuse application";

    int ret;
    struct fuse_args args = FUSE_ARGS_INIT(argc, argv);

    /* Set defaults -- we have to use strdup so that
       fuse_opt_parse can free the defaults if other
       values are specified */
    options.filename = strdup("hello");
    options.contents = strdup("Hello World!\n");

    /* Parse options */
    if (fuse_opt_parse(&args, &options, option_spec, NULL) == -1) {
        return 1;
    }

    /* When --help is specified, first print our own file-system
       specific help text, then signal fuse_main to show
       additional help (by adding `--help` to the options again)
       without usage: line (by setting argv[0] to the empty
       string) */
    if (options.show_help) {
        show_help(argv[0]);
        assert(fuse_opt_add_arg(&args, "--help") == 0);
        args.argv[0][0] = '\0';
    }

    ret = fuse_main(args.argc, args.argv, &hello_oper, NULL);
    fuse_opt_free_args(&args);
    return ret;
}
