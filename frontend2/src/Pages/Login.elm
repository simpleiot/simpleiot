module Pages.Login exposing (Model, Msg, page)

import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Generated.Params as Params
import Global
import Spa.Page
import Utils.Spa exposing (Page)


white =
    Element.rgb 1 1 1


grey =
    Element.rgb 0.9 0.9 0.9


blue =
    Element.rgb 0 0 0.8


red =
    Element.rgb 0.8 0 0


darkBlue =
    Element.rgb 0 0 0.9


page : Page Params.Login Model Msg model msg appMsg
page =
    Spa.Page.component
        { title = always "Login"
        , init = always init
        , update = always update
        , subscriptions = always subscriptions
        , view = always view
        }



-- INIT


type alias Model =
    { email : String
    , password : String
    }


init : Params.Login -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ =
    ( { email = "", password = "" }
    , Cmd.none
    , Cmd.none
    )



-- UPDATE


type Msg
    = Update Model
    | Login


update : Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update msg model =
    case msg of
        Update m ->
            ( m, Cmd.none, Cmd.none )

        Login ->
            ( model, Cmd.none, Spa.Page.send (Global.Login model) )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW

loginForm : Model -> Element Msg
loginForm model =
    column
        [ width (px 400)
        , spacing 12
        , centerX
        , centerY
        , spacing 36
        , padding 10
        , height shrink
        ]
        [ Input.email
            [ spacing 12 ]
            { text = model.email
            , placeholder = Just (Input.placeholder [] (text "email"))
            , onChange = \new -> Update { model | email = new }
            , label = Input.labelAbove [ Font.size 14 ] (text "Username")
            }
        , Input.currentPassword
            [ spacing 12 ]
            { text = model.password
            , placeholder = Just (Input.placeholder [] (text "password"))
            , onChange = \new -> Update { model | password = new }
            , label = Input.labelAbove [ Font.size 14 ] (text "Password")
            , show = False
            }
        , Input.button
            [ Background.color blue
            , Font.color white
            , Border.color darkBlue
            , paddingXY 32 16
            , Border.rounded 3
            , width (px 200)
            ]
            { onPress = Just Login
            , label = Element.text "Login"
            }
        ]

view : Model -> Element Msg
view model =
    loginForm model
