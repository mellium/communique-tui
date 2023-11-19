#ifndef GST_H
#define GST_H

#include <gst/gst.h>

void gstreamer_init();
GstElement *gstreamer_receive_create_pipeline(char *pipeline);
void gstreamer_receive_start_pipeline(GstElement *pipeline);
void gstreamer_receive_stop_pipeline(GstElement *pipeline);

#endif