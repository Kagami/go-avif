#include <stdlib.h>
#include <string.h>
#include <assert.h>
#include <aom/aom_encoder.h>
#include <aom/aomcx.h>
#include "av1.h"

#define SET_CODEC_CONTROL(ctrl, val) \
  {if (aom_codec_control(ctx, ctrl, val)) return AVIF_ERROR_CODEC_INIT;}

typedef struct {
  aom_img_fmt_t fmt;
  int dst_c_dec_h;
  int dst_c_dec_v;
  int bps;
  int bytes_per_sample;
} avif_format;

static avif_format convert_subsampling(const avif_subsampling subsampling) {
  avif_format fmt = { 0 };
  switch (subsampling) {
  case AVIF_SUBSAMPLING_I420:
    fmt.fmt = AOM_IMG_FMT_I420;
    fmt.dst_c_dec_h = 2;
    fmt.dst_c_dec_v = 2;
    fmt.bps = 12;
    fmt.bytes_per_sample = 1;
    break;
  default:
    assert(0);
  }
  return fmt;
}

// We don't use aom_img_wrap() because it forces padding for odd picture
// sizes (c) libaom/common/y4minput.c
static void convert_frame(const avif_frame *frame, aom_image_t *aom_frame) {
  memset(aom_frame, 0, sizeof(*aom_frame));
  avif_format fmt = convert_subsampling(frame->subsampling);
  aom_frame->fmt = fmt.fmt;
  aom_frame->w = aom_frame->d_w = frame->width;
  aom_frame->h = aom_frame->d_h = frame->height;
  aom_frame->x_chroma_shift = fmt.dst_c_dec_h >> 1;
  aom_frame->y_chroma_shift = fmt.dst_c_dec_v >> 1;
  aom_frame->bps = fmt.bps;
  int pic_sz = frame->width * frame->height * fmt.bytes_per_sample;
  int c_w = (frame->width + fmt.dst_c_dec_h - 1) / fmt.dst_c_dec_h;
  c_w *= fmt.bytes_per_sample;
  int c_h = (frame->height + fmt.dst_c_dec_v - 1) / fmt.dst_c_dec_v;
  int c_sz = c_w * c_h;
  aom_frame->stride[AOM_PLANE_Y] = frame->width * fmt.bytes_per_sample;
  aom_frame->stride[AOM_PLANE_U] = aom_frame->stride[AOM_PLANE_V] = c_w;
  aom_frame->planes[AOM_PLANE_Y] = frame->data;
  aom_frame->planes[AOM_PLANE_U] = frame->data + pic_sz;
  aom_frame->planes[AOM_PLANE_V] = frame->data + pic_sz + c_sz;
}

static int get_frame_stats(aom_codec_ctx_t *ctx,
                           const aom_image_t *frame,
                           aom_fixed_buf_t *stats) {
  if (aom_codec_encode(ctx, frame, 1/*pts*/, 1/*duration*/, 0/*flags*/))
    return AVIF_ERROR_FRAME_ENCODE;

  const aom_codec_cx_pkt_t *pkt = NULL;
  aom_codec_iter_t iter = NULL;
  int got_pkts = 0;
  while ((pkt = aom_codec_get_cx_data(ctx, &iter)) != NULL) {
    got_pkts = 1;
    if (pkt->kind == AOM_CODEC_STATS_PKT) {
      const uint8_t *const pkt_buf = pkt->data.twopass_stats.buf;
      const size_t pkt_size = pkt->data.twopass_stats.sz;
      stats->buf = realloc(stats->buf, stats->sz + pkt_size);
      memcpy((uint8_t *)stats->buf + stats->sz, pkt_buf, pkt_size);
      stats->sz += pkt_size;
    }
  }
  return got_pkts;
}

static int encode_frame(aom_codec_ctx_t *ctx,
                        const aom_image_t *frame,
                        avif_buffer *obu) {
  if (aom_codec_encode(ctx, frame, 1/*pts*/, 1/*duration*/, 0/*flags*/))
    return AVIF_ERROR_FRAME_ENCODE;

  const aom_codec_cx_pkt_t *pkt = NULL;
  aom_codec_iter_t iter = NULL;
  int got_pkts = 0;
  while ((pkt = aom_codec_get_cx_data(ctx, &iter)) != NULL) {
    got_pkts = 1;
    if (pkt->kind == AOM_CODEC_CX_FRAME_PKT) {
      const uint8_t *const pkt_buf = pkt->data.frame.buf;
      const size_t pkt_size = pkt->data.frame.sz;
      obu->buf = realloc(obu->buf, obu->sz + pkt_size);
      memcpy((uint8_t *)obu->buf + obu->sz, pkt_buf, pkt_size);
      obu->sz += pkt_size;
    }
  }
  return got_pkts;
}

static avif_error init_codec(aom_codec_iface_t *iface,
                             aom_codec_ctx_t *ctx,
                             const aom_codec_enc_cfg_t *aom_cfg,
                             const avif_config *cfg) {
  if (aom_codec_enc_init(ctx, iface, aom_cfg, 0))
    return AVIF_ERROR_CODEC_INIT;

  SET_CODEC_CONTROL(AOME_SET_CPUUSED, cfg->speed)
  SET_CODEC_CONTROL(AOME_SET_CQ_LEVEL, cfg->quality)
  if (cfg->quality == 0) {
    SET_CODEC_CONTROL(AV1E_SET_LOSSLESS, 1)
  }
  SET_CODEC_CONTROL(AV1E_SET_TILE_COLUMNS, 1)
  SET_CODEC_CONTROL(AV1E_SET_TILE_ROWS, 1)
  SET_CODEC_CONTROL(AV1E_SET_ROW_MT, 1)
  SET_CODEC_CONTROL(AV1E_SET_FRAME_PARALLEL_DECODING, 0)

  return AVIF_OK;
}

static avif_error do_pass1(aom_codec_ctx_t *ctx,
                           const aom_image_t *frame,
                           aom_fixed_buf_t *stats) {
  avif_error res = AVIF_OK;

  // Calculate frame statistics.
  if ((res = get_frame_stats(ctx, frame, stats)) < 0)
    goto fail;

  // Flush encoder.
  while ((res = get_frame_stats(ctx, NULL, stats)) > 0)
    continue;

fail:
  return res < 0 ? res : AVIF_OK;
}

static avif_error do_pass2(aom_codec_ctx_t *ctx,
                           const aom_image_t *frame,
                           avif_buffer *obu) {
  avif_error res = AVIF_OK;

  // Encode frame.
  if ((res = encode_frame(ctx, frame, obu)) < 0)
    goto fail;

  // Flush encoder.
  while ((res = encode_frame(ctx, NULL, obu)) > 0)
    continue;

fail:
  return res < 0 ? res : AVIF_OK;
}

avif_error avif_encode_frame(const avif_config *cfg,
                             const avif_frame *frame,
                             avif_buffer *obu) {
  // Validation.
  assert(cfg->threads >= 1);
  assert(cfg->speed >= AVIF_MIN_SPEED && cfg->speed <= AVIF_MAX_SPEED);
  assert(cfg->quality >= AVIF_MIN_QUALITY && cfg->quality <= AVIF_MAX_QUALITY);
  assert(frame->width && frame->height);

  // Prepare image.
  aom_image_t aom_frame;
  convert_frame(frame, &aom_frame);

  // Setup codec.
  avif_error res = AVIF_OK;
  aom_codec_ctx_t codec;
  aom_fixed_buf_t stats = { NULL, 0 };
  aom_codec_iface_t *iface = aom_codec_av1_cx();
  aom_codec_enc_cfg_t aom_cfg;
  if (aom_codec_enc_config_default(iface, &aom_cfg, 0)) {
    res = AVIF_ERROR_CODEC_INIT;
    goto fail;
  }
  aom_cfg.g_limit = 1;
  aom_cfg.g_w = frame->width;
  aom_cfg.g_h = frame->height;
  aom_cfg.g_timebase.num = 1;
  aom_cfg.g_timebase.den = 24;
  aom_cfg.rc_end_usage = AOM_Q;
  aom_cfg.g_threads = cfg->threads;

  // Pass 1.
  aom_cfg.g_pass = AOM_RC_FIRST_PASS;
  if ((res = init_codec(iface, &codec, &aom_cfg, cfg)))
    goto fail;
  if ((res = do_pass1(&codec, &aom_frame, &stats)))
    goto fail;
  if (aom_codec_destroy(&codec)) {
    res = AVIF_ERROR_CODEC_DESTROY;
    goto fail;
  }

  // Pass 2.
  aom_cfg.g_pass = AOM_RC_LAST_PASS;
  aom_cfg.rc_twopass_stats_in = stats;
  if ((res = init_codec(iface, &codec, &aom_cfg, cfg)))
    goto fail;
  if ((res = do_pass2(&codec, &aom_frame, obu)))
    goto fail;
  if (aom_codec_destroy(&codec)) {
    res = AVIF_ERROR_CODEC_DESTROY;
    goto fail;
  }

fail:
  free(stats.buf);
  return res;
}
