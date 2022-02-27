/* GStreamer
 *
 * Copyright (C) 2015 Sebastian Dröge <sebastian@centricular.com>
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Library General Public
 * License as published by the Free Software Foundation; either
 * version 2 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Library General Public License for more details.
 *
 * You should have received a copy of the GNU Library General Public
 * License along with this library; if not, write to the
 * Free Software Foundation, Inc., 51 Franklin St, Fifth Floor,
 * Boston, MA 02110-1301, USA.
 */

#ifndef __GST_PLAY_TYPES_H__
#define __GST_PLAY_TYPES_H__

#include <gst/gst.h>
#include <gst/play/play-prelude.h>

G_BEGIN_DECLS

/**
 * GstPlay:
 * Since: 1.20
 */
typedef struct _GstPlay GstPlay;
typedef struct _GstPlayClass GstPlayClass;

/**
 * GstPlaySignalAdapter:
 * Since: 1.20
 */
typedef struct _GstPlaySignalAdapter GstPlaySignalAdapter;
typedef struct _GstPlaySignalAdapterClass GstPlaySignalAdapterClass;

G_END_DECLS

#endif /* __GST_PLAY_TYPES_H__ */


