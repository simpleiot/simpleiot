module Pages.SignIn exposing (Model, Msg, page)

import Api.Auth exposing (User)
import Api.Data exposing (Data)
import Effect exposing (Effect)
import Element exposing (..)
import Element.Font as Font
import Element.Input as Input
import Gen.Params.SignIn exposing (Params)
import Gen.Route as Route
import Page
import Request exposing (Request)
import Shared
import Storage exposing (Storage)
import UI.Form as Form
import UI.Style as Style
import Utils.Route
import View exposing (View)


page : Shared.Model -> Request.With Params -> Page.With Model Msg
page shared req =
    Page.advanced
        { init = init shared
        , update = update shared.storage
        , view = view
        , subscriptions = subscriptions
        }



-- INIT


type alias Model =
    { user : Data User
    , email : String
    , password : String
    , error : Maybe String
    }


init : Shared.Model -> ( Model, Effect Msg )
init shared =
    ( Model
        (case shared.storage.user of
            Just auth ->
                Api.Data.Success auth

            Nothing ->
                Api.Data.NotAsked
        )
        ""
        ""
        Nothing
    , Effect.none
    )



-- UPDATE


type Msg
    = EditEmail String
    | EditPass String
    | SignIn
    | GotUser (Data User)
    | NoOp


update : Storage -> Msg -> Model -> ( Model, Effect Msg )
update storage msg model =
    case msg of
        EditEmail email ->
            ( { model | email = String.toLower email }, Effect.none )

        EditPass password ->
            ( { model | password = password }, Effect.none )

        SignIn ->
            ( model
            , Effect.fromCmd <|
                Api.Auth.login
                    { user =
                        { email = model.email
                        , password = model.password
                        }
                    , onResponse = GotUser
                    }
            )

        NoOp ->
            ( model, Effect.none )

        GotUser user ->
            let
                error =
                    case user of
                        Api.Data.Success _ ->
                            Nothing

                        Api.Data.Failure _ ->
                            Just "Login Failure"

                        _ ->
                            Just "Login unknown state"
            in
            ( { model | user = user, error = error }
            , case Api.Data.toMaybe user of
                Just user_ ->
                    Effect.fromCmd <| Storage.signIn user_ storage

                Nothing ->
                    Effect.none
            )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW


view : Model -> View Msg
view model =
    { title = "SIOT SignIn"
    , attributes = []
    , element =
        el [ centerX, centerY, Form.onEnter SignIn ] <|
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
                            Form.button
                                { label = "Sign In"
                                , color = Style.colors.gray
                                , onPress = NoOp
                                }

                        else
                            Form.button
                                { label = "Sign In"
                                , color = Style.colors.blue
                                , onPress = SignIn
                                }
                    ]
                ]
    }
