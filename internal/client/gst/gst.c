#include "gst.h"

#include <gst/app/gstappsrc.h>

typedef struct SampleHandlerUserData {
  int pipelineId;
} SampleHandlerUserData;

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

void gstreamer_receive_push_buffer(GstElement *pipeline, void *buffer, int len) {
    GstElement *src = gst_bin_get_by_name(GST_BIN(pipeline), "src");
    if (src != NULL) {
        gpointer p = g_memdup(buffer, len);
        GstBuffer *buffer = gst_buffer_new_wrapped(p, len);
        gst_app_src_push_buffer(GST_APP_SRC(src), buffer);
        gst_object_unref(src);
    }
}

GstFlowReturn gstreamer_send_new_sample_handler(GstElement *object, gpointer user_data) {
    GstSample *sample = NULL;
    GstBuffer *buffer = NULL;
    gpointer copy = NULL;
    gsize copy_size = 0;
    SampleHandlerUserData *s = (SampleHandlerUserData *)user_data;

    g_signal_emit_by_name (object, "pull-sample", &sample);
    if (sample) {
        buffer = gst_sample_get_buffer(sample);
        if (buffer) {
        gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
        goHandlePipelineBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), s->pipelineId);
        }
        gst_sample_unref (sample);
    }

    return GST_FLOW_OK;
}

GstElement *gstreamer_send_create_pipeline(char *pipeline) {
    GError *error = NULL;
    return gst_parse_launch(pipeline, &error);
}

void gstreamer_send_start_pipeline(GstElement *pipeline, int pipelineId) {
    SampleHandlerUserData *s = calloc(1, sizeof(SampleHandlerUserData));
    s->pipelineId = pipelineId;

    GstElement *appsink = gst_bin_get_by_name(GST_BIN(pipeline), "appsink");
    g_object_set(appsink, "emit-signals", TRUE, NULL);
    g_signal_connect(appsink, "new-sample", G_CALLBACK(gstreamer_send_new_sample_handler), s);
    gst_object_unref(appsink);

    gst_element_set_state(pipeline, GST_STATE_PLAYING);
    GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
    GstMessage *msg = gst_bus_timed_pop_filtered(bus, GST_CLOCK_TIME_NONE, GST_MESSAGE_ERROR | GST_MESSAGE_EOS);

    gst_object_unref(bus);
    gst_message_unref (msg);
}

void gstreamer_send_stop_pipeline(GstElement *pipeline) {
    gst_element_set_state(pipeline, GST_STATE_NULL);
}

void gstreamer_free_pipeline(GstElement *pipeline) {
    gst_object_unref(pipeline);
}