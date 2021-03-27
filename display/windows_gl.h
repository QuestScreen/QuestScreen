
#include <SDL2/SDL.h>

PFNGLCREATESHADERPROC _glCreateShader;
#define glCreateShader(_a) (*_glCreateShader)(_a)

PFNGLSHADERSOURCEPROC _glShaderSource;
#define glShaderSource(_a, _b, _c, _d) (*_glShaderSource)(_a, _b, _c, _d)

PFNGLCOMPILESHADERPROC _glCompileShader;
#define glCompileShader(_a) (*_glCompileShader)(_a)

PFNGLGETSHADERIVPROC _glGetShaderiv;
#define glGetShaderiv(_a, _b, _c) (*_glGetShaderiv)(_a, _b, _c)

PFNGLGETSHADERINFOLOGPROC _glGetShaderInfoLog;
#define glGetShaderInfoLog(_a, _b, _c, _d) (*_glGetShaderInfoLog)(_a, _b, _c, _d)

PFNGLDELETESHADERPROC _glDeleteShader;
#define glDeleteShader(_a) (*_glDeleteShader)(_a)

PFNGLCREATEPROGRAMPROC _glCreateProgram;
#define glCreateProgram() (*_glCreateProgram)()

PFNGLATTACHSHADERPROC _glAttachShader;
#define glAttachShader(_a, _b) (*_glAttachShader)(_a, _b)

PFNGLLINKPROGRAMPROC _glLinkProgram;
#define glLinkProgram(_a) (*_glLinkProgram)(_a)

PFNGLGETPROGRAMIVPROC _glGetProgramiv;
#define glGetProgramiv(_a, _b, _c) (*_glGetProgramiv)(_a, _b, _c)

PFNGLGETPROGRAMINFOLOGPROC _glGetProgramInfoLog;
#define glGetProgramInfoLog(_a, _b, _c, _d) (*_glGetProgramInfoLog)(_a, _b, _c, _d)

PFNGLDELETEPROGRAMPROC _glDeleteProgram;
#define glDeleteProgram(_a) (*_glDeleteProgram)(_a)

PFNGLGENBUFFERSPROC _glGenBuffers;
#define glGenBuffers(_a, _b) (*_glGenBuffers)(_a, _b)

PFNGLBINDBUFFERPROC _glBindBuffer;
#define glBindBuffer(_a, _b) (*_glBindBuffer)(_a, _b)

PFNGLBUFFERDATAPROC _glBufferData;
#define glBufferData(_a, _b, _c, _d) (*_glBufferData)(_a, _b, _c, _d)

PFNGLDELETEBUFFERSPROC _glDeleteBuffers;
#define glDeleteBuffers(_a, _b) (*_glDeleteBuffers)(_a, _b)

PFNGLGETUNIFORMLOCATIONPROC _glGetUniformLocation;
#define glGetUniformLocation(_a, _b) (*_glGetUniformLocation)(_a, _b)

PFNGLGETATTRIBLOCATIONPROC _glGetAttribLocation;
#define glGetAttribLocation(_a, _b) (*_glGetAttribLocation)(_a, _b)

PFNGLUSEPROGRAMPROC _glUseProgram;
#define glUseProgram(_a) (*_glUseProgram)(_a)

PFNGLVERTEXATTRIBPOINTERPROC _glVertexAttribPointer;
#define glVertexAttribPointer(_a, _b, _c, _d, _e, _f) (*_glVertexAttribPointer)(_a, _b, _c, _d, _e, _f)

PFNGLENABLEVERTEXATTRIBARRAYPROC _glEnableVertexAttribArray;
#define glEnableVertexAttribArray(_a) (*_glEnableVertexAttribArray)(_a)

PFNGLACTIVETEXTUREPROC _glActiveTexture;
#define glActiveTexture(_a) (*_glActiveTexture)(_a)

PFNGLUNIFORM1IPROC _glUniform1i;
#define glUniform1i(_a, _b) (*_glUniform1i)(_a, _b)

PFNGLUNIFORM1FPROC _glUniform1f;
#define glUniform1f(_a, _b) (*_glUniform1f)(_a, _b)

PFNGLUNIFORM2FVPROC _glUniform2fv;
#define glUniform2fv(_a, _b, _c) (*_glUniform2fv)(_a, _b, _c)

PFNGLUNIFORM4FPROC _glUniform4f;
#define glUniform4f(_a, _b, _c, _d, _e) (*_glUniform4f)(_a, _b, _c, _d, _e)

PFNGLGENFRAMEBUFFERSPROC _glGenFramebuffers;
#define glGenFramebuffers(_a, _b) (*_glGenFramebuffers)(_a, _b)

PFNGLBINDFRAMEBUFFERPROC _glBindFramebuffer;
#define glBindFramebuffer(_a, _b) (*_glBindFramebuffer)(_a, _b)

PFNGLFRAMEBUFFERTEXTURE2DPROC _glFramebufferTexture2D;
#define glFramebufferTexture2D(_a, _b, _c, _d, _e) (*_glFramebufferTexture2D)(_a, _b, _c, _d, _e)

PFNGLCHECKFRAMEBUFFERSTATUSPROC _glCheckFramebufferStatus;
#define glCheckFramebufferStatus(_a) (*_glCheckFramebufferStatus)(_a)

PFNGLDELETEFRAMEBUFFERSPROC _glDeleteFramebuffers;
#define glDeleteFramebuffers(_a, _b) (*_glDeleteFramebuffers)(_a, _b)