package types

import "context"

//
//   The bunch of interfaces are structure skeleton to build a extensible protocol stream architecture.
//
//   In mosn, we have 4 layers to build a mesh, stream is the inheritance layer to bond protocol layer and proxy layer together.
//	 -----------------------
//   |        PROXY          |
//    -----------------------
//   |       STREAMING       |
//    -----------------------
//   |        PROTOCOL       |
//    -----------------------
//   |         NET/IO        |
//    -----------------------
//
//   Core model in stream layer is stream, which manages process of a request and a corresponding response.
// 	 Event listeners can be installed into a stream to monitor event.
//	 Stream has two related models, encoder and decoder:
// 		- StreamEncoder: a sender encodes request/response to binary and sends it out, flag 'endStream' means data is ready to sendout, no need to wait for further input.
//		- StreamDecoder: It's more like a decode listener to get called on a receiver receives binary and decodes to a request/response.
//	 	- Stream does not have a predetermined direction, so StreamEncoder could be a request encoder as a client or a response encode as a server. It's just about the usecase, so does StreamDecoder.
//
//   Stream:
//   	- Event listener
// 			- StreamEventListener
//      - Encoder
// 			- StreamEncoder
// 		- Decoder
//			- StreamDecoder
//
//	 In order to meet the expansion requirements in the stream processing, StreamEncoderFilters and StreamDecoderFilters are introduced as a filter chain in encode/decode process.
//   Filter's method will be called on corresponding stream process stage and returns a status(Continue/Stop) to effect the control flow.
//
//   From an abstract perspective, stream represents a virtual process on underlying connection. To make stream interactive with connection, some intermediate object can be used.
//	 StreamConnection is the core model to connect connection system to stream system. As a example, when proxy reads binary data from connection, it dispatches data to StreamConnection to do protocol decode.
//   Specifically, ClientStreamConnection uses a NewStream to exchange StreamDecoder with StreamEncoder.
//   Engine provides a callbacks(StreamEncoderFilterCallbacks/StreamDecoderFilterCallbacks) to let filter interact with stream engine.
// 	 As a example, a encoder filter stopped the encode process, it can continue it by StreamEncoderFilterCallbacks.ContinueEncoding later. Actually, a filter engine is a encoder/decoder itself.
//
//   Below is the basic relation on stream and connection:
//    --------------------------------------------------------------------------------------------------------------
//   |																												|
//   | 	  EventListener   								EventListener												|
//   |        *|                                               |*													|
//   |         |                                               |													|
//   |        1|        1    1  	    		1     *        |1													|
// 	 |	    Connection -------- StreamConnection ---------- Stream													|
//   |        1|                   	   |1				   	   |1                                                   |
// 	 |		   |					   |				   	   |                                                    |
//	 |         |                   	   |					   |--------------------------------					|
//   |        *|                   	   |					   |*           	 				|*					|
//   |	 ConnectionFilter    		   |			 StreamEncoder[sender]  		StreamDecoder[receiver]			|
//   |								   |*					   |1				 				|1					|
// 	 |						StreamConnectionEventListener	   |				 				|					|
//	 |													       |*				 				|*					|
//	 |										 	 		StreamDecoderFilter	   			StreamDecoderFilter			|
//	 |													   	   |1								|1					|
//	 |													   	   |								|					|
// 	 |													       |1								|1					|
//	 |										 		StreamDecoderFilterCallbacks     StreamDecoderFilterCallbacks	|
//   |																												|
//    --------------------------------------------------------------------------------------------------------------
//

type StreamResetReason string

const (
	StreamConnectionTermination StreamResetReason = "ConnectionTermination"
	StreamConnectionFailed      StreamResetReason = "ConnectionFailed"
	StreamLocalReset            StreamResetReason = "StreamLocalReset"
	StreamOverflow              StreamResetReason = "StreamOverflow"
	StreamRemoteReset           StreamResetReason = "StreamRemoteReset"
)

// Core model in stream layer, a generic protocol stream
type Stream interface {
	// Add stream event listener
	AddEventListener(streamEventListener StreamEventListener)

	// Remove stream event listener
	RemoveEventListener(streamEventListener StreamEventListener)

	// Reset stream. Any registered StreamEventListener.OnResetStream should be called.
	ResetStream(reason StreamResetReason)

	// Enable/disable further stream data
	ReadDisable(disable bool)
}

// Stream event listener
type StreamEventListener interface {
	// Called on a stream is been reset
	OnResetStream(reason StreamResetReason)

	// Called when a stream , or the connection the stream is sending to, goes over its high watermark.
	OnAboveWriteBufferHighWatermark()

	// Called when a stream, or the connection the stream is sending to, goes from over its high watermark to under its low watermark
	OnBelowWriteBufferLowWatermark()
}

// Encode protocol stream
type StreamEncoder interface {
	// Encode headers
	// endStream supplies whether this is a header only request/response
	EncodeHeaders(headers interface{}, endStream bool) error

	// Encode data
	// endStream supplies whether this is the last data frame
	EncodeData(data IoBuffer, endStream bool) error

	// Encode trailers, implicitly ends the stream.
	EncodeTrailers(trailers map[string]string) error

	// Get related stream
	GetStream() Stream
}

// Listeners called on decode stream event
type StreamDecoder interface {
	// Called with decoded headers
	// endStream supplies whether this is a header only request/response
	OnDecodeHeaders(headers map[string]string, endStream bool)

	// Called with a decoded data
	// endStream supplies whether this is the last data
	OnDecodeData(data IoBuffer, endStream bool)

	// Called with a decoded trailers frame, implicitly ends the stream.
	OnDecodeTrailers(trailers map[string]string)
}

// A connection runs multiple streams
type StreamConnection interface {
	// Dispatch incoming data
	Dispatch(buffer IoBuffer)

	// Protocol on the connection
	Protocol() Protocol

	// Send go away to remote for graceful shutdown
	GoAway()

	// Called when the underlying Connection goes over its high watermark.
	OnUnderlyingConnectionAboveWriteBufferHighWatermark()

	// Called when the underlying Connection goes from over its high watermark to under its low watermark.
	OnUnderlyingConnectionBelowWriteBufferLowWatermark()
}

// A server side stream connection.
type ServerStreamConnection interface {
	StreamConnection
}

// A client side stream connection.
type ClientStreamConnection interface {
	StreamConnection

	// Create a new outgoing request stream
	// responseDecoder supplies the decoder listeners on decode event
	// StreamEncoder supplies the encoder to write the request
	NewStream(streamId string, responseDecoder StreamDecoder) StreamEncoder
}

// Stream connection event listener
type StreamConnectionEventListener interface {
	// Called on remote sends 'go away'
	OnGoAway()
}

// Stream connection event listener for server connection
type ServerStreamConnectionEventListener interface {
	StreamConnectionEventListener

	// return request stream decoder
	NewStream(streamId string, responseEncoder StreamEncoder) StreamDecoder
}

type StreamFilterBase interface {
	OnDestroy()
}

// Called by stream filter to interact with underlying stream
type StreamFilterCallbacks interface {
	// the originating connection
	Connection() Connection

	// Reset the underlying stream
	ResetStream()

	// Route for current stream
	Route() Route

	// Get stream id
	StreamId() string

	// Request info related to the stream
	RequestInfo() RequestInfo
}

type StreamEncoderFilter interface {
	StreamFilterBase

	EncodeHeaders(headers interface{}, endStream bool) FilterHeadersStatus

	EncodeData(buf IoBuffer, endStream bool) FilterDataStatus

	EncodeTrailers(trailers map[string]string) FilterTrailersStatus

	SetEncoderFilterCallbacks(cb StreamEncoderFilterCallbacks)
}

type StreamEncoderFilterCallbacks interface {
	StreamFilterCallbacks

	ContinueEncoding()

	EncodingBuffer() IoBuffer

	AddEncodedData(buf IoBuffer, streamingFilter bool)

	OnEncoderFilterAboveWriteBufferHighWatermark()

	OnEncoderFilterBelowWriteBufferLowWatermark()

	SetEncoderBufferLimit(limit uint32)

	EncoderBufferLimit() uint32
}

type StreamDecoderFilter interface {
	StreamFilterBase

	DecodeHeaders(headers map[string]string, endStream bool) FilterHeadersStatus

	DecodeData(buf IoBuffer, endStream bool) FilterDataStatus

	DecodeTrailers(trailers map[string]string) FilterTrailersStatus

	SetDecoderFilterCallbacks(cb StreamDecoderFilterCallbacks)
}

type StreamDecoderFilterCallbacks interface {
	StreamFilterCallbacks

	ContinueDecoding()

	DecodingBuffer() IoBuffer

	AddDecodedData(buf IoBuffer, streamingFilter bool)

	EncodeHeaders(headers interface{}, endStream bool)

	EncodeData(buf IoBuffer, endStream bool)

	EncodeTrailers(trailers map[string]string)

	OnDecoderFilterAboveWriteBufferHighWatermark()

	OnDecoderFilterBelowWriteBufferLowWatermark()

	AddDownstreamWatermarkCallbacks(cb DownstreamWatermarkEventListener)

	RemoveDownstreamWatermarkCallbacks(cb DownstreamWatermarkEventListener)

	SetDecoderBufferLimit(limit uint32)

	DecoderBufferLimit() uint32
}

type DownstreamWatermarkEventListener interface {
	OnAboveWriteBufferHighWatermark()

	OnBelowWriteBufferLowWatermark()
}

type StreamFilterChainFactory interface {
	CreateFilterChain(context context.Context, callbacks FilterChainFactoryCallbacks)
}

type FilterChainFactoryCallbacks interface {
	AddStreamDecoderFilter(filter StreamDecoderFilter)

	AddStreamEncoderFilter(filter StreamEncoderFilter)
}

type FilterHeadersStatus string

const (
	FilterHeadersStatusContinue      FilterHeadersStatus = "Continue"
	FilterHeadersStatusStopIteration FilterHeadersStatus = "StopIteration"
)

type FilterDataStatus string

const (
	FilterDataStatusContinue                  FilterDataStatus = "Continue"
	FilterDataStatusStopIterationAndBuffer    FilterDataStatus = "StopIterationAndBuffer"
	FilterDataStatusStopIterationAndWatermark FilterDataStatus = "StopIterationAndWatermark"
	FilterDataStatusStopIterationNoBuffer     FilterDataStatus = "StopIterationNoBuffer"
)

type FilterTrailersStatus string

const (
	FilterTrailersStatusContinue      FilterTrailersStatus = "Continue"
	FilterTrailersStatusStopIteration FilterTrailersStatus = "StopIteration"
)
