#include "renderer.h"
#include <stdio.h>
#include <stdlib.h>

static const char *image_vshader_src =
    "#version 150                \n"
    "uniform mat3x2 u_transform; \n"
    "in vec2 a_position;         \n"
    "out vec2 v_texCoord;        \n"
    "void main()  {              \n"
    "  gl_Position = vec4(u_transform*vec3(a_position,1), 0, 1); \n"
    "  v_texCoord = vec2(a_position.x, 1-a_position.y);  \n"
    "}";

// TODO: alpha blending
static const char *image_fshader_src =
    "#version 150                   \n"
    "precision mediump float;       \n"
    "in vec2 v_texCoord;            \n"
    "out vec4 fragColor;            \n"
    "uniform sampler2D s_texture;   \n"
    "void main()  {                 \n"
    "  fragColor = texture(s_texture, v_texCoord); \n" // -> texture2D, gl_FragColor
    "}";

static const char *rect_vshader_src =
    "#version 150                \n"
    "uniform mat3x2 u_transform; \n"
    "in vec2 a_position;         \n"
    "void main() {               \n"
    "  gl_Position = vec4(u_transform*vec3(a_position, 1), 0, 1);\n"
    //"  gl_Position = vec4(a_position, 0, 1);\n"
    "}";

// TODO: alpha blending
static const char *rect_fshader_src =
    "#version 150              \n"
    "precision mediump float;  \n"
    "uniform vec4 u_color;     \n"
    "out vec4 fragColor;       \n"
    "void main() {             \n"
    "  fragColor = u_color;    \n"
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

  glGenVertexArrays(1, &e->vao);
  glBindVertexArray(e->vao);

  e->canvas_count = 0;

  if ((e->image.id = link_program(image_vshader_src, image_fshader_src)) == 0) {
    return false;
  }
  safeGetLocation(Uniform, image, transform, "u_transform");
  safeGetLocation(Attrib, image, position, "a_position");
  safeGetLocation(Uniform, image, texture, "s_texture");

  if ((e->rect.id = link_program(rect_vshader_src, rect_fshader_src)) == 0) {
    glDeleteProgram(e->image.id);
    return false;
  }
  safeGetLocation(Uniform, rect, transform, "u_transform");
  safeGetLocation(Attrib, rect, position, "a_position");
  safeGetLocation(Uniform, rect, color, "u_color");

  glDisable(GL_DEPTH_TEST);

  glEnable(GL_BLEND);
  glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);

  glDepthMask(false);

  return true;
}

void engine_close(engine_t *e) {
  glDeleteBuffers(1, &e->vbo);
  glDeleteVertexArrays(1, &e->vao);
}

uint32_t gen_texture(engine_t *e, GLenum format, GLint bytesPerPixel,
    GLsizei w, GLsizei h, void *pixels) {
  GLuint ret;
  (void)e;
  glGenTextures(1, &ret);
  glBindTexture(GL_TEXTURE_2D, ret);
  glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
  glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
  if (pixels != NULL) {
    glPixelStorei(GL_UNPACK_ALIGNMENT, bytesPerPixel);
  }
  glTexImage2D(GL_TEXTURE_2D, 0, format, w, h, 0, format, GL_UNSIGNED_BYTE,
      pixels);
  return ret;
}

void draw_image(
    engine_t *e, GLuint texture, float transform[6], uint8_t alpha) {
  glBindBuffer(GL_ARRAY_BUFFER, e->vbo);
  glBindVertexArray(e->vao);
  glUseProgram(e->image.id);

  glVertexAttribPointer(e->image.position, 2, GL_FLOAT,
      GL_FALSE, 2 * sizeof(GLfloat), 0);
  glEnableVertexAttribArray(e->image.position);

  glActiveTexture(GL_TEXTURE0);
  glBindTexture(GL_TEXTURE_2D, texture);
  glUniform1i(e->image.texture, 0);
  glUniformMatrix3x2fv(e->image.transform, 1, GL_FALSE, transform);

  glDrawArrays(GL_TRIANGLE_FAN, 0, 4);
}

void draw_rect(engine_t *e, float transform[6],
    uint8_t r, uint8_t g, uint8_t b, uint8_t a) {

  //debug_matrix(transform);

  glBindBuffer(GL_ARRAY_BUFFER, e->vbo);
  glUseProgram(e->rect.id);

  glVertexAttribPointer(e->rect.position, 2, GL_FLOAT,
      GL_FALSE, 2 * sizeof(GLfloat), 0);
  glEnableVertexAttribArray(e->rect.position);

  glUniform4f(e->rect.color, (float)r / 255.0, (float)g / 255.0,
      (float)b / 255.0, (float)a / 255.0);
  glUniformMatrix3x2fv(e->rect.transform, 1, GL_FALSE, transform);

  glDrawArrays(GL_TRIANGLE_FAN, 0, 4);
}

#define renderFbFailure(_name)\
  case _name:\
    puts("unable to create canvas: returned " #_name);\
    break

void create_canvas(engine_t *e, GLsizei w, GLsizei h,
    GLuint *oldFb, GLuint *targetFb, GLuint *targetTex) {
  *targetTex = gen_texture(e, GL_RGBA, 4, w, h, NULL);

  GLint tmpOldFb;
  glGetIntegerv(GL_DRAW_FRAMEBUFFER_BINDING, &tmpOldFb);
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
    renderFbFailure(GL_FRAMEBUFFER_UNDEFINED);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_ATTACHMENT);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_MISSING_ATTACHMENT);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_DRAW_BUFFER);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_READ_BUFFER);
    renderFbFailure(GL_FRAMEBUFFER_UNSUPPORTED);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_MULTISAMPLE);
    renderFbFailure(GL_FRAMEBUFFER_INCOMPLETE_LAYER_TARGETS);
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