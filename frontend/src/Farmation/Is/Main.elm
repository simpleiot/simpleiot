port module Main exposing (Msg(..), main, update, view)

import Browser
import Html exposing (Html, button, div, text)
import Html.Events exposing (onClick)
import Farmation.Is.Lcd as Lcd
import Time


main =
    Browser.document
        { init = init
        , update = update
        , view = view
        , subscriptions = subscriptions
        }



-- Subscriptions


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Time.every 1000 Tick
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
    = SetPixel Int Int Bool
    | Tick Time.Posix


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        SetPixel x y v ->
            ( { model
                | lcdData =
                    Lcd.setPixel x y v model.lcdData
              }
            , Cmd.none
            )

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
