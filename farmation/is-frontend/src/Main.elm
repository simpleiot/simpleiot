module Main exposing (Model, Msg(..), init, main, update, view)

import Browser
import Html exposing (Html, button, div, text)
import Html.Events exposing (onClick)
import Lcd exposing (..)


main =
    Browser.sandbox { init = init, update = update, view = view }



-- MODEL


type alias Model =
    { lcdData : LcdData
    }


init : Model
init =
    { lcdData = lcdData
    }



-- UPDATE


type Msg
    = SetPixel Int Int Bool


update : Msg -> Model -> Model
update msg model =
    case msg of
        SetPixel x y v ->
            { model
                | lcdData =
                    Lcd.setPixel x y v model.lcdData
            }



-- VIEW


view : Model -> Html Msg
view model =
    div []
        [ lcd model.lcdData
        ]
