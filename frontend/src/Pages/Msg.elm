module Pages.Msg exposing (Model, Msg, Params, page)

import Api.Auth as Auth exposing (Auth)
import Api.Data as Data exposing (Data)
import Api.Msg
import Api.Response exposing (Response)
import Element exposing (..)
import Element.Input as Input
import Http
import Shared
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Spa.Page as Page exposing (Page)
import Spa.Url exposing (Url)
import Time
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
    { auth : Auth
    , now : Time.Posix
    , msg : String
    , error : Maybe String
    }


defaultModel : Model
defaultModel =
    Model
        Auth.empty
        (Time.millisToPosix 0)
        ""
        Nothing


init : Shared.Model -> Url Params -> ( Model, Cmd Msg )
init shared _ =
    case shared.auth of
        Just auth ->
            let
                model =
                    { defaultModel | auth = auth }
            in
            ( model
            , Cmd.none
            )

        Nothing ->
            ( defaultModel
            , Utils.Route.navigate shared.key Route.SignIn
            )



-- UPDATE


type Msg
    = UpdateMsg String
    | ApiSend
    | ApiRespSend (Data Response)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        UpdateMsg message ->
            ( { model | msg = message }, Cmd.none )

        ApiSend ->
            ( model
            , Api.Msg.send
                { token = model.auth.token
                , msg = model.msg
                , now = model.now
                , onResponse = ApiRespSend
                }
            )

        ApiRespSend resp ->
            case resp of
                Data.Success _ ->
                    ( { model | msg = "" }, Cmd.none )

                Data.Failure err ->
                    ( popError "Error sending msg" err model, Cmd.none )

                _ ->
                    ( model, Cmd.none )


popError : String -> Http.Error -> Model -> Model
popError desc err model =
    { model | error = Just (desc ++ ": " ++ Data.errorToString err) }


save : Model -> Shared.Model -> Shared.Model
save model shared =
    { shared
        | error =
            case model.error of
                Nothing ->
                    shared.error

                Just _ ->
                    model.error
        , lastError =
            case model.error of
                Nothing ->
                    shared.lastError

                Just _ ->
                    shared.now
    }


load : Shared.Model -> Model -> ( Model, Cmd Msg )
load shared model =
    ( { model | now = shared.now, error = Nothing }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none



-- VIEW


view : Model -> Document Msg
view model =
    { title = "Send Message to all users"
    , body =
        [ column
            [ width fill, spacing 32 ]
            [ Input.multiline [ width fill ]
                { onChange = UpdateMsg
                , text = model.msg
                , placeholder = Nothing
                , label = Input.labelAbove [] <| text "Message to send:"
                , spellcheck = True
                }
            , Form.button
                { label = "send now"
                , color = Style.colors.blue
                , onPress = ApiSend
                }
            ]
        ]
    }
