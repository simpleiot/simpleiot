port module Main exposing (Msg(..), main, update, view)

import Browser
import Farmation.Is.Lcd as Lcd
import Html exposing (Html, button, div, text)
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



-- Subscriptions


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Time.every 1000 Tick
        , portIn (pixValueDecoder >> SetPixel)
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
    = SetPixel (Result Json.Decode.Error Pix)
    | Tick Time.Posix


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        SetPixel result ->
            case result of
                Ok pix ->
                    ( { model
                        | lcdData =
                            Lcd.setPixel pix.x pix.y pix.v model.lcdData
                      }
                    , Cmd.none
                    )

                Err err ->
                    let
                        _ =
                            Debug.log "Pixel decode error: " err
                    in
                    ( model, Cmd.none )

        Tick _ ->
            ( model, Cmd.none )



-- VIEW


view : Model -> Browser.Document Msg
view model =
    { title = "Injector • Sentry"
    , body =
        [ div []
            [ Lcd.lcd model.lcdData
            ]
        ]
    }



-- Misc functions/data structures


type alias Pix =
    { x : Int
    , y : Int
    , v : Bool
    }


pixDecoder : Json.Decode.Decoder Pix
pixDecoder =
    Json.Decode.map3 Pix
        (Json.Decode.field "x" Json.Decode.int)
        (Json.Decode.field "y" Json.Decode.int)
        (Json.Decode.field "v" Json.Decode.bool)


pixValueDecoder : Json.Decode.Value -> Result Json.Decode.Error Pix
pixValueDecoder v =
    Json.Decode.decodeValue pixDecoder v
