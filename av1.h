#pragma once

#include <stdint.h>

enum {
  AVIF_MIN_SPEED = 0,
  AVIF_MAX_SPEED = 8,
  AVIF_MIN_QUALITY = 0,
  AVIF_MAX_QUALITY = 63,
};

typedef enum {
  AVIF_OK = 0,
  AVIF_ERROR_GENERAL = -1000,
  AVIF_ERROR_CODEC_INIT,
  AVIF_ERROR_CODEC_DESTROY,
  AVIF_ERROR_FRAME_ENCODE,
} avif_error;

typedef enum {
  AVIF_SUBSAMPLING_I420,
} avif_subsampling;

typedef struct {
  int threads;
  int speed;
  int quality;
} avif_config;

typedef struct {
  uint16_t width;
  uint16_t height;
  avif_subsampling subsampling;
  uint8_t *data;
} avif_frame;

typedef struct {
  void *buf;
  size_t sz;
} avif_buffer;

avif_error avif_encode_frame(const avif_config *cfg,
                             const avif_frame *frame,
                             avif_buffer *obu);
