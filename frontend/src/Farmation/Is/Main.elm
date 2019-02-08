port module Main exposing (Msg(..), main, update, view)

import Array
import Browser
import Farmation.Is.Lcd as Lcd
import Html exposing (Html, button, div, h2, map, text)
import Html.Events exposing (onClick)
import Json.Decode
import Json.Encode
import Time


main =
    Browser.document
        { init = init
        , update = update
        , view = view
        , subscriptions = subscriptions
        }



-- ports


port portIn : (Json.Decode.Value -> msg) -> Sub msg


port portOut : Json.Encode.Value -> Cmd msg



-- Subscriptions


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Time.every 1000 Tick
        , portIn (portValueDecoder >> ProcessPortValue)
        ]



-- MODEL


type alias Model =
    { lcdData : Lcd.Data
    }


init : () -> ( Model, Cmd Msg )
init model =
    ( { lcdData = Lcd.defaultData
      }
    , Cmd.none
    )



-- UPDATE


type Msg
    = ProcessPortValue (Result Json.Decode.Error PortValue)
    | Tick Time.Posix
    | GotLcdMsg Lcd.Msg


type alias KeyPressMsg =
    { msgType : String
    , key : String
    }


encodeKeyPressMsg : KeyPressMsg -> Json.Encode.Value
encodeKeyPressMsg msg =
    Json.Encode.object
        [ ( "msgType", Json.Encode.string <| msg.msgType )
        , ( "key", Json.Encode.string <| msg.key )
        ]


keyToKeyPressMsg : Lcd.Key -> KeyPressMsg
keyToKeyPressMsg key =
    let
        keyString =
            case key of
                Lcd.KeyUp ->
                    "KeyUp"

                Lcd.KeyDown ->
                    "KeyDown"

                Lcd.KeyRight ->
                    "KeyRight"

                Lcd.KeyLeft ->
                    "KeyLeft"

                Lcd.KeyEnter ->
                    "KeyEnter"

                Lcd.KeySK1 ->
                    "KeySK1"

                Lcd.KeySK2 ->
                    "KeySK2"

                Lcd.KeySK3 ->
                    "KeySK3"

                Lcd.KeySK4 ->
                    "KeySK4"
    in
    { msgType = "key"
    , key = keyString
    }


processPortValue : PortValue -> Model -> ( Model, Cmd Msg )
processPortValue portValue model =
    case portValue of
        PixelValue pix ->
            ( { model
                | lcdData =
                    Lcd.setPixel pix.x pix.y pix.v model.lcdData
              }
            , Cmd.none
            )

        BltSolidValue blt ->
            ( { model
                | lcdData =
                    Lcd.setSolidBlock blt.x blt.y blt.w blt.h blt.v model.lcdData
              }
            , Cmd.none
            )

        BltValue blt ->
            ( { model
                | lcdData =
                    Lcd.setBlock blt.x blt.y blt.w blt.h blt.data model.lcdData
              }
            , Cmd.none
            )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        ProcessPortValue result ->
            case result of
                Ok portValue ->
                    processPortValue portValue model

                Err err ->
                    let
                        _ =
                            Debug.log "Port value decode error: " err
                    in
                    ( model, Cmd.none )

        GotLcdMsg lcdMsg ->
            case lcdMsg of
                Lcd.GotKey key ->
                    ( model
                    , keyToKeyPressMsg key
                        |> encodeKeyPressMsg
                        |> portOut
                    )

        Tick _ ->
            ( model, Cmd.none )



-- VIEW


view : Model -> Browser.Document Msg
view model =
    { title = "Injector • Sentry"
    , body =
        [ div []
            [ h2 [] [ text "Injector Sentry" ]
            , map GotLcdMsg (Lcd.lcd model.lcdData)
            ]
        ]
    }



-- Misc functions/data structures


type alias Pixel =
    { x : Int
    , y : Int
    , v : Bool
    }


pixelDecoder : Json.Decode.Decoder Pixel
pixelDecoder =
    Json.Decode.map3 Pixel
        (Json.Decode.field "x" Json.Decode.int)
        (Json.Decode.field "y" Json.Decode.int)
        (Json.Decode.field "v" Json.Decode.bool)


type alias BltSolid =
    { x : Int
    , y : Int
    , w : Int
    , h : Int
    , v : Bool
    }


bltSolidDecoder : Json.Decode.Decoder BltSolid
bltSolidDecoder =
    Json.Decode.map5 BltSolid
        (Json.Decode.field "x" Json.Decode.int)
        (Json.Decode.field "y" Json.Decode.int)
        (Json.Decode.field "w" Json.Decode.int)
        (Json.Decode.field "h" Json.Decode.int)
        (Json.Decode.field "v" Json.Decode.bool)


type alias Blt =
    { x : Int
    , y : Int
    , w : Int
    , h : Int
    , data : Array.Array Bool
    }


bltDecoder : Json.Decode.Decoder Blt
bltDecoder =
    Json.Decode.map5 Blt
        (Json.Decode.field "x" Json.Decode.int)
        (Json.Decode.field "y" Json.Decode.int)
        (Json.Decode.field "w" Json.Decode.int)
        (Json.Decode.field "h" Json.Decode.int)
        (Json.Decode.field "data" (Json.Decode.array Json.Decode.bool))


type PortValue
    = PixelValue Pixel
    | BltSolidValue BltSolid
    | BltValue Blt


portDecoder : Json.Decode.Decoder PortValue
portDecoder =
    Json.Decode.oneOf
        [ Json.Decode.map BltSolidValue bltSolidDecoder
        , Json.Decode.map PixelValue pixelDecoder
        , Json.Decode.map BltValue bltDecoder
        ]


pixValueDecoder : Json.Decode.Value -> Result Json.Decode.Error Pixel
pixValueDecoder v =
    Json.Decode.decodeValue pixelDecoder v


portValueDecoder : Json.Decode.Value -> Result Json.Decode.Error PortValue
portValueDecoder v =
    Json.Decode.decodeValue portDecoder v
