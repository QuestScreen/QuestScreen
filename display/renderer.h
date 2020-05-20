#if __APPLE__
#define GL_SILENCE_DEPRECATION
#include <OpenGL/gl3.h>
#else
#include <SDL2/SDL_opengles2.h>
#endif

#include <stdbool.h>

typedef struct {
  struct {
    GLuint id;
    GLint transform, position, texture;
  } image;
  struct {
    GLuint id;
    GLint transform, position, color;
  } rect;
  GLuint vao, vbo;
  uint8_t canvas_count;
} engine_t;

bool engine_init(engine_t *e);

void engine_close(engine_t *e);

uint32_t gen_texture(engine_t *e, GLenum format, GLsizei w, GLsizei h,
    void *pixels);

void draw_image(engine_t *e, GLuint texture, float  transform[6],
    uint8_t alpha, bool texHasAlpha);

void draw_rect(
  engine_t *e, float transform[6], uint8_t color[4], bool copyAlpha);

void create_canvas(engine_t *e, GLsizei w, GLsizei h,
    GLuint *oldFb, GLuint *targetFb, GLuint *targetTex, bool withAlpha);
void destroy_canvas(
  engine_t *e, GLuint targetFb, GLuint targetTex, GLuint prevFb);
void finish_canvas(engine_t *e, GLuint targetFb, GLuint prevFb);

uint8_t canvas_count(engine_t *e);