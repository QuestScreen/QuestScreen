#include "renderer.h"
#include <stdio.h>
#include <stdlib.h>

#define mat_mult(_m, _v) "vec2(" #_m "[0].x * " #_v ".x + " #_m "[1].x * " \
    #_v ".y + " #_m "[2].x, " #_m "[0].y * " #_v ".x + " #_m "[1].y * " \
    #_v ".y + " #_m "[2].y)"

#ifdef __APPLE__
#define glslversion "#version 150\n"
#define texture(_s, _v) "texture(" #_s ", " #_v ")"
#define fragColor "fragColor"
#else
#define glslversion "#version 100\n"
#define texture(_s, _v) "texture2D(" #_s ", " #_v ")"
#define fragColor "gl_FragColor"
#endif

static const char *image_vshader_src = glslversion
    "uniform vec2 u_transform[3];\n"
#ifdef __APPLE__
    "in vec2 a_position;         \n"
    "out vec2 v_texCoord;        \n"
#else
    "attribute vec2 a_position;  \n"
    "varying vec2 v_texCoord;    \n"
#endif
    "void main()  {              \n"
    "  gl_Position = vec4(" mat_mult(u_transform, a_position) ", 0, 1); \n"
    "  v_texCoord = vec2(a_position.x, 1.0-a_position.y);  \n"
    "}";

static const char *image_fshader_src = glslversion
    "precision mediump float;       \n"
#ifdef __APPLE__
    "in vec2 v_texCoord;            \n"
    "out vec4 fragColor;            \n"
#else
    "varying vec2 v_texCoord;       \n"
#endif
    "uniform sampler2D s_texture;   \n"
    "uniform float u_alpha;         \n"
    "void main()  {                 \n"
    "  vec4 c = " texture(s_texture, v_texCoord) "; \n"
    "  " fragColor " = vec4(c.rgb, u_alpha * c.a);  \n"
    "}";

static const char *masking_vshader_src = glslversion
    "uniform vec2 u_posTrans[3];\n"
    "uniform vec2 u_texTrans[3];\n"
#ifdef __APPLE__
    "in vec2 a_position;        \n"
    "out vec2 v_texCoord;       \n"
#else
    "attribute vec2 a_position; \n"
    "varying vec2 v_texCoord;   \n"
#endif
    "void main()  {             \n"
    "  gl_Position = vec4(" mat_mult(u_posTrans, a_position) ", 0, 1); \n"
    "  vec2 flipped = vec2(a_position.x, 1.0-a_position.y);            \n"
    "  v_texCoord = " mat_mult(u_texTrans, flipped) "; \n"
    "}";

static const char *masking_fshader_src = glslversion
    "precision mediump float;       \n"
#ifdef __APPLE__
    "in vec2 v_texCoord;            \n"
    "out vec4 fragColor;            \n"
#else
    "varying vec2 v_texCoord;       \n"
#endif
    "uniform sampler2D s_texture;   \n"
    "uniform vec4 u_primary;        \n"
    "uniform vec4 u_secondary;      \n"
    "void main()  {                 \n"
    "  float a = " texture(s_texture, v_texCoord) ".r;      \n"
    "  " fragColor " = a * u_primary + (1.0-a) * u_secondary; \n"
    "}";

static const char *rect_vshader_src = glslversion
    "uniform vec2 u_transform[3];\n"
#ifdef __APPLE__
    "in vec2 a_position;         \n"
#else
    "attribute vec2 a_position;  \n"
#endif
    "void main() {               \n"
    "  gl_Position = vec4(" mat_mult(u_transform, a_position) ", 0, 1);\n"
    "}";

static const char *rect_fshader_src = glslversion
    "precision mediump float;  \n"
    "uniform vec4 u_color;     \n"
#ifdef __APPLE__
    "out vec4 fragColor;       \n"
#endif
    "void main() {             \n"
    "  " fragColor "= u_color; \n"
    "}";

GLuint load_shader(const char *shaderSrc, GLenum type) {
  GLuint shader;
  GLint compiled;
  shader = glCreateShader(type);
  if(shader == 0) {
    puts("unable to glCreateShader!");
    return 0;
  }
  glShaderSource(shader, 1, &shaderSrc, NULL);
  glCompileShader(shader);
  glGetShaderiv(shader, GL_COMPILE_STATUS, &compiled);
  if(!compiled) {
    GLint infoLen = 0;
    glGetShaderiv(shader, GL_INFO_LOG_LENGTH, &infoLen);
    if(infoLen > 1) {
      char* infoLog = malloc(sizeof(char) * infoLen);
      glGetShaderInfoLog(shader, infoLen, NULL, infoLog);
      puts("Error compiling shader:");
      puts("-----------------------");
      puts(shaderSrc);
      puts("-----------------------");
      puts(infoLog);
      free(infoLog);
    } else puts("Unknown problem compiling shader!");
    glDeleteShader(shader);
    return 0;
  }
  return shader;
}

static void debug_matrix(float transform[6]) {
  float x1 = transform[4];
  float y1 = transform[5];
  float x2 = 1.0*transform[0] + 1.0*transform[2] + transform[4];
  float y2 = 1.0*transform[1] + 1.0*transform[3] + transform[5];

  printf("resulting rectangle: (%f, %f) -- (%f, %f)\n", x1, y1, x2, y2);
}

#define error_case(_name) case _name: fputs("  " #_name "\n", stderr); break

static void debug_check_error(int line, const char *lastCall) {
  GLenum err = glGetError();
  if (err != GL_NO_ERROR) {
    fprintf(stderr, "renderer.c(%d): OpenGL Error during call to %s:\n",
        line, lastCall);
    switch (err) {
      error_case(GL_INVALID_ENUM);
      error_case(GL_INVALID_VALUE);
      error_case(GL_INVALID_OPERATION);
      error_case(GL_OUT_OF_MEMORY);
      error_case(GL_INVALID_FRAMEBUFFER_OPERATION);
      default: fputs("  unknown GL error!\n", stderr);
    }
  }
}

#define check_error(_lastCall) debug_check_error(__LINE__, _lastCall)

#define ensure_load_shader(_name, _type, _src) \
  if ((_name = load_shader(_src, _type)) == 0) return 0

static GLuint link_program(const char *vsrc, const char *fsrc) {
  GLuint vertexShader, fragmentShader, ret;
  GLint linked;
  ensure_load_shader(vertexShader, GL_VERTEX_SHADER, vsrc);
  ensure_load_shader(fragmentShader, GL_FRAGMENT_SHADER, fsrc);
  if ((ret = glCreateProgram()) == 0) {
    puts("unable to glCreateProgram!");
    return 0;
  }
  glAttachShader(ret, vertexShader);
  glAttachShader(ret, fragmentShader);
  glLinkProgram(ret);

  glGetProgramiv(ret, GL_LINK_STATUS, &linked);
  if(linked) {
    return ret;
  }
  GLint infoLen = 0;
  glGetProgramiv(ret, GL_INFO_LOG_LENGTH, &infoLen);
  if(infoLen > 1) {
    char* infoLog = malloc(sizeof(char) * infoLen);
    glGetProgramInfoLog(ret, infoLen, NULL, infoLog);
    puts("Error linking program:");
    puts(infoLog);
    free(infoLog);
  } else puts("Unknown problem linking program!");
  glDeleteProgram(ret);
  return 0;
}

#define safeGetLocation(_kind, _prog, _target, _nameStr) do {\
  e->_prog._target = glGet ## _kind ## Location(e->_prog.id, _nameStr);\
  if (e->_prog._target < 0) {\
    puts("failed to get attribute " _nameStr " of prog " #_prog);\
    return false;\
  }\
} while(false)

bool engine_init(engine_t *e) {
  GLfloat vertices[] = {0.0f, 0.0f, 1.0f, 0.0f,
                        1.0f, 1.0f, 0.0f, 1.0f};
  glGenBuffers(1, &e->vbo);
  glBindBuffer(GL_ARRAY_BUFFER, e->vbo);
  glBufferData(GL_ARRAY_BUFFER, sizeof(vertices), vertices, GL_STATIC_DRAW);

#ifdef __APPLE__
  glGenVertexArrays(1, &e->vao);
  glBindVertexArray(e->vao);
#endif

  e->canvas_count = 0;

  if ((e->image.id = link_program(image_vshader_src, image_fshader_src)) == 0) {
    return false;
  }
  safeGetLocation(Uniform, image, transform, "u_transform");
  safeGetLocation(Attrib, image, position, "a_position");
  safeGetLocation(Uniform, image, texture, "s_texture");
  safeGetLocation(Uniform, image, alpha, "u_alpha");

  if ((e->mask.id = link_program(masking_vshader_src, masking_fshader_src)) == 0) {
    return false;
  }
  safeGetLocation(Uniform, mask, posTrans, "u_posTrans");
  safeGetLocation(Uniform, mask, texTrans, "u_texTrans");
  safeGetLocation(Attrib, mask, position, "a_position");
  safeGetLocation(Uniform, mask, texture, "s_texture");
  safeGetLocation(Uniform, mask, primary, "u_primary");
  safeGetLocation(Uniform, mask, secondary, "u_secondary");

  if ((e->rect.id = link_program(rect_vshader_src, rect_fshader_src)) == 0) {
    glDeleteProgram(e->image.id);
    return false;
  }
  safeGetLocation(Uniform, rect, transform, "u_transform");
  safeGetLocation(Attrib, rect, position, "a_position");
  safeGetLocation(Uniform, rect, color, "u_color");

  glDisable(GL_DEPTH_TEST);
  glDepthMask(false);

  glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);

  glGetIntegerv(GL_MAX_TEXTURE_SIZE, &e->maxTexSize);
  return true;
}

void engine_close(engine_t *e) {
  glDeleteBuffers(1, &e->vbo);
#ifdef __APPLE__
  glDeleteVertexArrays(1, &e->vao);
#endif
}

uint32_t gen_texture(engine_t *e, GLenum format, GLsizei w, GLsizei h,
    void *pixels) {
  GLuint ret;
  (void)e;
  glGenTextures(1, &ret);
  glBindTexture(GL_TEXTURE_2D, ret);
  glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
  glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
  // only used for mask, no reason to set something else for others.
  glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_REPEAT);
  glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_REPEAT);
  if (pixels != NULL) {
    switch (format) {
      case GL_RGBA:
        glPixelStorei(GL_UNPACK_ALIGNMENT, 4);
        break;
      case GL_RGB:
        glPixelStorei(GL_UNPACK_ALIGNMENT, 3);
        break;
      case GL_SINGLE_VALUE_COLOR:
        glPixelStorei(GL_UNPACK_ALIGNMENT, 1);
        break;
      default:
        fputs("error: unsupported texture format!\n", stderr);
        break;
    }

  }
  glTexImage2D(GL_TEXTURE_2D, 0, format, w, h, 0, format, GL_UNSIGNED_BYTE,
      pixels);
  return ret;
}

void draw_image(
    engine_t *e, GLuint texture, float transform[6], uint8_t alpha,
    bool texHasAlpha) {
  if ( alpha != 255 || texHasAlpha) {
    glEnable(GL_BLEND);
  }

  glBindBuffer(GL_ARRAY_BUFFER, e->vbo);
#ifdef __APPLE__
  glBindVertexArray(e->vao);
#endif
  glUseProgram(e->image.id);

  glVertexAttribPointer(e->image.position, 2, GL_FLOAT,
      GL_FALSE, 2 * sizeof(GLfloat), 0);
  glEnableVertexAttribArray(e->image.position);

  glActiveTexture(GL_TEXTURE0);
  glBindTexture(GL_TEXTURE_2D, texture);
  glUniform1i(e->image.texture, 0);
  glUniform1f(e->image.alpha, (float)alpha / 255.0f);
  glUniform2fv(e->image.transform, 3, transform);

  glDrawArrays(GL_TRIANGLE_FAN, 0, 4);

  if ( alpha != 255 || texHasAlpha) {
    glDisable(GL_BLEND);
  }
}

void setUniformColor(GLint id, uint8_t color[4]) {
  glUniform4f(id, (float)color[0] / 255.0, (float)color[1] / 255.0,
      (float)color[2] / 255.0, (float)color[3] / 255.0);
}

void draw_masked(engine_t *e, GLuint texture, float posTrans[6],
    float texTrans[6], uint8_t primary[4], uint8_t secondary[4]) {
  glBindBuffer(GL_ARRAY_BUFFER, e->vbo);
#ifdef __APPLE__
  glBindVertexArray(e->vao);
#endif
  glUseProgram(e->mask.id);

  glVertexAttribPointer(e->mask.position, 2, GL_FLOAT,
      GL_FALSE, 2 * sizeof(GLfloat), 0);
  glEnableVertexAttribArray(e->mask.position);

  glActiveTexture(GL_TEXTURE0);
  glBindTexture(GL_TEXTURE_2D, texture);
  glUniform1i(e->mask.texture, 0);
  glUniform2fv(e->mask.posTrans, 3, posTrans);
  glUniform2fv(e->mask.texTrans, 3, texTrans);
  setUniformColor(e->mask.primary, primary);
  setUniformColor(e->mask.secondary, secondary);

  glDrawArrays(GL_TRIANGLE_FAN, 0, 4);
}

void draw_rect(
    engine_t *e, float transform[6], uint8_t color[4], bool copyAlpha) {

  //debug_matrix(transform);

  if (!copyAlpha && color[3] != 255) {
    glEnable(GL_BLEND);
  }

  glBindBuffer(GL_ARRAY_BUFFER, e->vbo);
  glUseProgram(e->rect.id);

  glVertexAttribPointer(e->rect.position, 2, GL_FLOAT,
      GL_FALSE, 2 * sizeof(GLfloat), 0);
  glEnableVertexAttribArray(e->rect.position);

  setUniformColor(e->rect.color, color);
  glUniform2fv(e->rect.transform, 3, transform);

  glDrawArrays(GL_TRIANGLE_FAN, 0, 4);

  glDisable(GL_BLEND);
}

#define renderFbFailure(_name)\
  case _name:\
    puts("unable to create canvas: returned " #_name);\
    break

void create_canvas(engine_t *e, GLsizei w, GLsizei h,
    GLuint *oldFb, GLuint *targetFb, GLuint *targetTex, bool withAlpha) {
  *targetTex = gen_texture(e, withAlpha ? GL_RGBA : GL_RGB, w, h, NULL);

  GLint tmpOldFb;
#ifdef __APPLE__
  glGetIntegerv(GL_DRAW_FRAMEBUFFER_BINDING, &tmpOldFb);
#else
  glGetIntegerv(GL_FRAMEBUFFER_BINDING, &tmpOldFb);
#endif
  *oldFb = (GLuint)tmpOldFb;

  glGenFramebuffers(1, targetFb);
  glBindFramebuffer(GL_FRAMEBUFFER, *targetFb);
  glFramebufferTexture2D(
      GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, *targetTex, 0);

#ifdef __APPLE__
  GLenum db[1] = { GL_COLOR_ATTACHMENT0 };
  glDrawBuffers(1, db);
#endif

  switch (glCheckFramebufferStatus(GL_FRAMEBUFFER)) {
    case GL_FRAMEBUFFER_COMPLETE:
      ++e->canvas_count;
      glClearColor(.0f, .0f, .0f, 1.0f);
		  glClear(GL_COLOR_BUFFER_BIT);
      return;
#ifdef __APPLE__
    renderFbFailure(GL_FRAMEBUFFER_UNDEFINED);
#endif
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_ATTACHMENT);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_MISSING_ATTACHMENT);
#ifdef __APPLE__
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_DRAW_BUFFER);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_READ_BUFFER);
#endif
    renderFbFailure(GL_FRAMEBUFFER_UNSUPPORTED);
#ifdef __APPLE__
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_MULTISAMPLE);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_LAYER_TARGETS);
#endif
  }
  glBindFramebuffer(GL_FRAMEBUFFER, *oldFb);
  glDeleteTextures(1, targetTex);
  glDeleteFramebuffers(1, targetFb);
  *targetFb = 0;
}

void destroy_canvas(
    engine_t *e, GLuint targetFb, GLuint targetTex, GLuint prevFb) {
  --e->canvas_count;
  glBindFramebuffer(GL_FRAMEBUFFER, prevFb);
  glDeleteFramebuffers(1, &targetFb);
  glDeleteTextures(1, &targetTex);
}

void finish_canvas(engine_t *e, GLuint targetFb, GLuint prevFb) {
  --e->canvas_count;
  glBindFramebuffer(GL_FRAMEBUFFER, prevFb);
  glDeleteFramebuffers(1, &targetFb);
}

uint8_t canvas_count(engine_t *e) {
  return e->canvas_count;
}
