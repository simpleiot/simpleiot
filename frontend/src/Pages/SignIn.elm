module Pages.SignIn exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Api.Data exposing (Data)
import Browser.Navigation exposing (Key)
import Element exposing (..)
import Element.Font as Font
import Element.Input as Input
import Shared
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Spa.Page as Page exposing (Page)
import Spa.Url as Url exposing (Url)
import UI.Form as Form
import UI.Style as Style
import Utils.Route


page : Page Params Model Msg
page =
    Page.application
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        , save = save
        , load = load
        }



-- INIT


type alias Params =
    ()


type alias Model =
    { auth : Data Auth
    , key : Key
    , email : String
    , password : String
    }


init : Shared.Model -> Url Params -> ( Model, Cmd Msg )
init shared { key } =
    ( Model
        (case shared.auth of
            Just auth ->
                Api.Data.Success auth

            Nothing ->
                Api.Data.NotAsked
        )
        key
        ""
        ""
    , Cmd.none
    )



-- UPDATE


type Msg
    = EditEmail String
    | EditPass String
    | SignIn
    | GotUser (Data Auth)
    | NoOp


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        EditEmail email ->
            ( { model | email = email }, Cmd.none )

        EditPass password ->
            ( { model | password = password }, Cmd.none )

        SignIn ->
            ( model
            , Api.Auth.login
                { user =
                    { email = model.email
                    , password = model.password
                    }
                , onResponse = GotUser
                }
            )

        NoOp ->
            ( model, Cmd.none )

        GotUser auth ->
            ( { model | auth = auth }
            , case Api.Data.toMaybe auth of
                Just auth_ ->
                    Utils.Route.navigate model.key Route.Top

                Nothing ->
                    Cmd.none
            )


save : Model -> Shared.Model -> Shared.Model
save model shared =
    { shared
        | auth =
            case Api.Data.toMaybe model.auth of
                Just auth ->
                    Just auth

                Nothing ->
                    shared.auth
    }


load : Shared.Model -> Model -> ( Model, Cmd Msg )
load _ model =
    ( model, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none



-- VIEW


view : Model -> Document Msg
view model =
    { title = "SIOT SignIn"
    , body =
        [ el [ centerX, centerY ] <|
            column
                [ spacing 32 ]
                [ el [ Font.size 24, Font.semiBold ]
                    (text "Sign in")
                , column [ spacing 16 ]
                    [ Input.email
                        []
                        { onChange = \e -> EditEmail e
                        , text = model.email
                        , placeholder = Just <| Input.placeholder [] <| text "email"
                        , label = Input.labelAbove [] <| text "Email"
                        }
                    , Input.newPassword
                        []
                        { onChange = \p -> EditPass p
                        , show = False
                        , text = model.password
                        , placeholder = Just <| Input.placeholder [] <| text "password"
                        , label = Input.labelAbove [] <| text "Password"
                        }
                    , el [ alignRight ] <|
                        if String.isEmpty model.email then
                            Form.button "Sign In" Style.colors.gray NoOp

                        else
                            Form.button "Sign In" Style.colors.blue SignIn
                    ]
                ]
        ]
    }
