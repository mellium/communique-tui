#include "gst.h"

#include <gst/app/gstappsrc.h>

void gstreamer_init() {
    gst_init(NULL, NULL);
}

GstElement *gstreamer_receive_create_pipeline(char *pipeline) {
    GError *error = NULL;
    return gst_parse_launch(pipeline, &error);
}

void gstreamer_receive_start_pipeline(GstElement *pipeline) {
    gst_element_set_state(pipeline, GST_STATE_PLAYING);
    GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
    GstMessage *msg = gst_bus_timed_pop_filtered(bus, GST_CLOCK_TIME_NONE, GST_MESSAGE_ERROR | GST_MESSAGE_EOS);
    // if (GST_MESSAGE_TYPE (msg) == GST_MESSAGE_ERROR) {
    //     g_error("An error occurred! Re-run with the GST_DEBUG=*:WARN environment variable set for more details.");
    // }
    gst_object_unref(bus);
    gst_message_unref (msg);
}

void gstreamer_receive_stop_pipeline(GstElement *pipeline) {
    gst_element_set_state(pipeline, GST_STATE_NULL);
}